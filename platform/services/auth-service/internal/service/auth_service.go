package service

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/crypto"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/pkg/security"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/http"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	repo "github.com/zerodayz7/platform/services/auth-service/internal/repository"
)

// region interface
type AuthService interface {
	AttemptLogin(ctx context.Context, email string, password []byte, fingerprint string) (*http.LoginResponse, error)
	Register(username, email, rawPassword string) (*model.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, newPassword string) error
	Verify2FA(ctx context.Context, token string, code []byte, fingerprint string, ip string) (*http.Verify2FAResponse, error)
	Resend2FACode(ctx context.Context, email string, token string) error
	Logout(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, fingerprint string) error
	RegisterDevice(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, clientIP string, req schemas.RegisterDeviceRequest) (*http.RegisterDeviceResponse, error)
	RefreshToken(ctx context.Context, tokenStr string, fingerprint string) (*http.RefreshResponse, error)

	VerifyDeviceSignature(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, signature, fingerprint string) (*http.LoginResponse, error)
	UnpairDevice(ctx context.Context, userID uuid.UUID, deviceFingerprint string, sessionID uuid.UUID, req schemas.UnpairDeviceRequest) error

	AttemptLoginStep2(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, signature, fingerprint, clientIP string) (*http.LoginResponse, error)
}

// region struct
type authService struct {
	userRepo     repo.UserRepository
	employeeRepo repo.EmployeeRepository
	refreshRepo  repo.RefreshTokenRepository
	cache        *redis.Cache
	cfg          *config.Config
}

// #region NewAuthService
func NewAuthService(
	userRepo repo.UserRepository,
	employeeRepo repo.EmployeeRepository,
	refreshRepo repo.RefreshTokenRepository,
	cache *redis.Cache,
	cfg *config.Config,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		employeeRepo: employeeRepo,
		refreshRepo:  refreshRepo,
		cache:        cache,
		cfg:          cfg,
	}
}

// #region AttemptLoginStep2
func (s *authService) AttemptLoginStep2(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, signature string, fingerprint string, clientIP string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	log.InfoMap("[AttemptLoginStep2] Rozpoczęcie weryfikacji Step 2", map[string]any{
		"user_id":    userID,
		"session_id": sessionID,
		"client_ip":  clientIP,
	})

	// 1. Pobranie wyzwania z Redisa (konwersja uuid na string dzieje się wewnątrz GetChallenge)
	storedChallenge, err := s.cache.GetChallenge(ctx, sessionID)
	if err != nil || storedChallenge == "" {
		log.WarnMap("[AttemptLoginStep2] Challenge nie znaleziony w Redis lub wygasł", map[string]any{
			"user_id": userID,
			"sid":     sessionID,
			"err":     err,
		})
		return nil, errors.ErrInvalidChallenge
	}

	// 2. Pobranie aktywnego poświadczenia pracownika
	cred, err := s.employeeRepo.GetActiveCredentialByUserID(ctx, userID)
	if err != nil || cred == nil {
		log.WarnMap("[AttemptLoginStep2] Nie znaleziono aktywnego poświadczenia w DB", map[string]any{
			"user_id": userID,
			"err":     err,
		})
		return nil, errors.ErrUntrustedDevice
	}

	if cred.ExpiresAt != nil && cred.ExpiresAt.Before(time.Now()) {
		log.WarnMap("[AttemptLoginStep2] Karta pracownika wygasła", map[string]any{
			"user_id":    userID,
			"card_sn":    cred.CardSerialNumber,
			"expires_at": cred.ExpiresAt,
		})
		return nil, errors.ErrUntrustedDevice
	}

	// 3. Weryfikacja podpisu cryptographic (Ed25519)
	challengeBytes := []byte(storedChallenge)
	if !shared.VerifyEd25519SignatureHex(cred.PublicKey, challengeBytes, signature) {
		log.WarnMap("SECURITY ALERT: Nieprawidłowy podpis pracownika", map[string]any{
			"user_id":       userID,
			"card_sn":       cred.CardSerialNumber,
			"pub_key_in_db": cred.PublicKey,
		})
		return nil, errors.ErrInvalidSignature
	}

	// 4. Usunięcie wyzwania z Redisa (Replay Attack Protection)
	_ = s.cache.DeleteChallenge(ctx, sessionID)

	// 5. Pobranie użytkownika
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		log.WarnMap("[AttemptLoginStep2] Nie znaleziono konta użytkownika", map[string]any{
			"user_id": userID,
			"err":     err,
		})
		return nil, errors.ErrUserNotFound
	}

	if err := s.CanUserLogin(user); err != nil {
		log.WarnMap("[AttemptLoginStep2] Konto zablokowane lub nieaktywne", map[string]any{
			"user_id": user.ID,
			"err":     err,
		})
		return nil, err
	}

	// 6. Pobranie metadanych profilu pracownika
	var permissions []string
	var instID, deptID, empNumber string

	if user.Role != model.RoleCitizen && user.Role != model.RoleUser {
		empProfile, err := s.employeeRepo.GetProfileByUserID(ctx, user.ID)
		if err == nil && empProfile != nil {
			permissions = empProfile.Permissions
			instID = empProfile.InstitutionID.String()
			deptID = empProfile.DepartmentID.String()
			empNumber = empProfile.EmployeeNumber
		}
	}

	// 7. Tworzenie tokenów i pełnej sesji w Redis
	accessToken, newSessionID, err := s.CreateAccessToken(ctx, user.ID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(user.ID, fingerprint, nil)
	if err != nil {
		return nil, errors.ErrInternal
	}

	sessionData := redis.UserSession{
		UserID:         user.ID.String(),
		Username:       user.Username,
		Email:          user.Email,
		Role:           string(user.Role),
		EmployeeNumber: empNumber,
		InstitutionID:  instID,
		DepartmentID:   deptID,
		Permissions:    permissions,
		Fingerprint:    crypto.HashSHA256(fingerprint),
		PublicKey:      cred.PublicKey,
	}

	if err := s.cache.SetSession(ctx, newSessionID, &sessionData, s.cfg.Session.TTL); err != nil {
		return nil, errors.ErrInternal
	}

	return &http.LoginResponse{
		Type:         "fullSuccess",
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		UserID:       user.ID.String(),
		ExpiresAt:    time.Now().Add(s.cfg.JWT.AccessTTL).Unix(),
	}, nil
}

// #region RegisterDevice
func (s *authService) RegisterDevice(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, clientIP string, req schemas.RegisterDeviceRequest) (*http.RegisterDeviceResponse, error) {
	log := shared.GetLogger()

	if sessionID == uuid.Nil {
		log.WarnMap("[RegisterDevice] Empty sessionID passed", map[string]any{"user_id": userID.String()})
		return nil, errors.ErrSessionExpired
	}

	// 1. Walidacja tymczasowej sesji parowania (Setup Session)
	setupSess, err := s.cache.GetSetupSession(ctx, sessionID)
	if err != nil || setupSess == nil {
		log.WarnMap("[RegisterDevice] Setup session expired or missing", map[string]any{"sid": sessionID})
		return nil, errors.ErrSessionExpired
	}

	if setupSess.Fingerprint != crypto.HashSHA256(req.DeviceFingerprint) {
		log.WarnMap("[RegisterDevice] Fingerprint mismatch", map[string]any{"sid": sessionID})
		return nil, errors.ErrInvalidDeviceFingerprint
	}

	// 2. Weryfikacja kryptograficzna wyzwania (Atomic challenge verification & deletion)
	pubKeyBytes, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
		log.ErrorObj("[RegisterDevice] Invalid public key format", err)
		return nil, errors.ErrInvalidPairingData
	}

	if err := s.verifyChallengeSession(ctx, sessionID, req.Signature, pubKeyBytes); err != nil {
		log.WarnMap("[RegisterDevice] Cryptographic verification failed", map[string]any{"sid": sessionID, "err": err})
		return nil, err
	}

	// Posprzątaj sesję setup po przejściu testu kryptograficznego
	_ = s.cache.DeleteSetupSession(ctx, sessionID)

	// 3. Pobranie użytkownika i weryfikacja stanu konta
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		log.ErrorObj("[RegisterDevice] User not found", err)
		return nil, errors.ErrUserNotFound
	}

	if err := s.CanUserLogin(user); err != nil {
		log.WarnMap("[RegisterDevice] Account locked or banned", map[string]any{"user_id": user.ID, "status": user.Status})
		return nil, err
	}

	// 4. Utworzenie lub aktualizacja rekordu urządzenia
	device := &model.UserDevice{
		UserID:              userID,
		DeviceFingerprint:   crypto.HashSHA256(req.DeviceFingerprint),
		PublicKey:           req.PublicKey,
		DeviceNameEncrypted: req.DeviceNameEncrypted,
		Platform:            req.Platform,
		IsVerified:          true,
		IsActive:            true,
		LastIP:              clientIP,
	}

	if err := s.userRepo.SaveDevice(ctx, device); err != nil {
		log.ErrorObj("[RegisterDevice] Failed to save device in DB", err)
		return nil, errors.ErrInternal
	}

	// 5. Generowanie poświadczeń (JWT Access & Refresh Token)
	accessToken, newSID, err := s.CreateAccessToken(ctx, userID, req.DeviceFingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(userID, req.DeviceFingerprint, &device.ID)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 6. Zapis pełnej sesji użytkownika w Redis (dla API Gateway)
	sessionData := s.buildUserSession(user, req.DeviceFingerprint, req.PublicKey)
	if err := s.cache.SetSession(ctx, newSID, &sessionData, s.cfg.Session.TTL); err != nil {
		log.ErrorObj("[RegisterDevice] Failed to persist session in Redis", err)
		return nil, errors.ErrInternal
	}

	log.InfoMap("✅ Device registered successfully", map[string]any{"user_id": userID.String(), "device_id": device.ID})

	return &http.RegisterDeviceResponse{
		Success:      true,
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		IsTrusted:    true,
		User: http.DeviceUserData{
			UserID:      user.ID.String(),
			Email:       user.Email,
			DisplayName: user.Username,
			LastLogin:   time.Now().Format(time.RFC3339),
			Roles:       []string{string(user.Role)},
		},
	}, nil
}

// region Logout
// #region Logout
func (s *authService) Logout(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, fingerprint string) error {
	log := shared.GetLogger()

	// 1. Pobierz sesję
	session, err := s.cache.GetSession(ctx, sessionID)
	if err != nil {
		log.WarnMap("Logout: session not found", map[string]any{"sid": sessionID})
		return errors.ErrUnauthorized
	}

	// 2. Weryfikacja bezpieczeństwa (UserID i opcjonalnie Fingerprint)
	if session.UserID != userID.String() || session.Fingerprint != fingerprint {
		log.ErrorMap("Logout security violation", map[string]any{
			"expected_uid": userID.String(),
			"actual_uid":   session.UserID,
			"expected_fpt": fingerprint,
			"actual_fpt":   session.Fingerprint,
		})
		return errors.ErrUnauthorized
	}

	// 3. Usuwanie sesji z Redis
	if err := s.cache.DeleteSession(ctx, sessionID); err != nil {
		return errors.ErrInternal
	}

	// 4. Unieważnienie Refresh Tokena w DB przy użyciu fingerprintu
	_ = s.refreshRepo.RevokeByFingerprint(ctx, userID, fingerprint)

	return nil
}

// region Verify2FA
// #region Verify2FA
func (s *authService) Verify2FA(ctx context.Context, token string, code []byte, fingerprint string, ip string) (*http.Verify2FAResponse, error) {
	log := shared.GetLogger()

	// Czyszczenie wrażliwego bufora kodu po zakończeniu funkcji
	defer clear(code)

	// 1. Pobieranie sesji 2FA z Cache
	session, err := s.cache.Get2FASession(ctx, token)
	if err != nil {
		return nil, errors.ErrInvalidCredentials
	}

	log.DebugInfo("2FA compare", map[string]any{
		"hash": session.CodeHash,
	})

	// 2. Weryfikacja kodu Argon2id z uwzględnieniem peppera
	valid, err := security.VerifyPassword(code, session.CodeHash, nil) // podmień s.pepper na nil jeśli nie masz pola w strukturze

	if err != nil || !valid {
		// Logika blokowania po błędnych próbach
		status, _ := s.cache.Verify2FAAttempt(ctx, token, 5, 5*time.Minute)
		log.DebugInfo("2FA verification failed", map[string]any{
			"status": status,
			"token":  token,
		})

		switch status {
		case "locked":
			return nil, errors.Err2FALocked
		default:
			return nil, errors.ErrInvalid2FACode
		}
	}

	// 3. Czyszczenie sesji 2FA
	_ = s.cache.Delete2FASession(ctx, token)

	// 4. Pobieranie użytkownika i aktualizacja metadanych logowania
	uid, _ := uuid.Parse(session.UserID)
	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, errors.ErrInternal
	}

	user.LastLogin = time.Now()
	user.LastIP = ip
	_ = s.userRepo.Update(ctx, user)

	setupToken, sessionID, err := s.CreateSetupToken(ctx, uid, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	err = s.cache.SetSetupSession(ctx, sessionID, &redis.UserSession{
		UserID:      session.UserID,
		Fingerprint: crypto.HashSHA256(fingerprint),
	}, s.cfg.Session.TTL)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 6. Generowanie Challenge (Ed25519)
	challenge, err := security.GenerateRandomString(32)
	if err != nil {
		log.ErrorObj("Failed to generate secure challenge", err)
		return nil, errors.ErrInternal
	}

	if err := s.cache.SetChallenge(ctx, sessionID, challenge, 5*time.Minute); err != nil {
		log.ErrorObj("Failed to save challenge in Redis", err)
		return nil, errors.ErrInternal
	}

	response := &http.Verify2FAResponse{
		Success:    true,
		SetupToken: setupToken,
		Challenge:  challenge,
		IsTrusted:  false,
	}

	// DEBUG INFO: Wypisujemy dokładnie to, co idzie do klienta
	log.DebugJSON("[DEBUG] Sending 2FA Response:", response)

	return response, nil
}

// #region Resend2FACode
func (s *authService) Resend2FACode(ctx context.Context, email string, token string) error {
	log := shared.GetLogger()

	// 1. Pobieramy istniejącą sesję 2FA
	session, err := s.cache.Get2FASession(ctx, token)
	if err != nil || session == nil {
		log.WarnMap("Resend2FA: session not found or expired", map[string]any{"token": token, "email": email})
		return errors.ErrSessionExpired
	}

	// 2. Weryfikacja zgodności adresu e-mail
	if !strings.EqualFold(session.Email, email) {
		log.WarnMap("SECURITY ALERT: Resend2FA email mismatch", map[string]any{
			"expected_email": session.Email,
			"provided_email": email,
		})
		return errors.ErrInvalidCredentials
	}

	// 3. Generujemy nowy bezpieczny kod OTP
	code, err := security.GenerateOTP(6)
	if err != nil {
		log.ErrorObj("Resend2FA: failed to generate OTP", err)
		return errors.ErrInternal
	}

	// 4. Hashujemy kod przed zapisem w pamięci podręcznej (przekazujemy string)
	codeBytes := []byte(code)
	defer clear(codeBytes)

	hashedCode, err := security.HashPassword(codeBytes, nil)
	if err != nil {
		log.ErrorObj("Resend2FA: failed to hash OTP", err)
		return errors.ErrInternal
	}

	// 5. Aktualizujemy podmieniony hash w sesji
	session.CodeHash = hashedCode

	// 6. Odświeżamy sesję 2FA w Redis (z resetem TTL na kolejne 5 minut)
	if err := s.cache.Set2FASession(ctx, token, *session, 5*time.Minute); err != nil {
		log.ErrorObj("Resend2FA: failed to update 2FA session in Redis", err)
		return errors.ErrInternal
	}

	log.DebugInfo("Resent 2FA code successfully", map[string]any{
		"email": email,
		"token": token,
		"code":  code,
	})

	return nil
}

// region AttemptLogin
// #region AttemptLogin
func (s *authService) AttemptLogin(ctx context.Context, email string, password []byte, fingerprint string) (*http.LoginResponse, error) {
	defer func() {
		if len(password) > 0 {
			for i := range password {
				password[i] = 0
			}
		}
	}()

	log := shared.GetLogger()

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.ErrInvalidCredentials
	}

	if err = s.CanUserLogin(user); err != nil {
		return nil, err
	}

	valid, err := security.VerifyPassword(password, user.Password, nil)
	if err != nil || !valid {
		return nil, s.handleFailedLogin(ctx, user.ID)
	}

	if user.FailedLoginAttempts > 0 {
		_ = s.userRepo.ResetFailedLoginAttempts(user.ID)
	}

	// 1. Odgałęzienie dla Urzędnika
	if user.Role != model.RoleCitizen && user.Role != model.RoleUser {
		log.Info("Procesowanie logowania urzędnika", map[string]any{"uid": user.ID, "role": user.Role})
		return s.prepareEmployeeLogin(ctx, user, fingerprint)
	}

	// 2. Odgałęzienie dla Zaufanego Urządzenia (Obywatel)
	device, err := s.userRepo.GetDeviceByFingerprint(ctx, user.ID, fingerprint)
	log.DebugDB("SCENARIUSZ A", device)

	if err == nil && device != nil && device.IsVerified && device.IsActive {
		return s.preparePreTrustSession(ctx, user, device.PublicKey, fingerprint)
	}

	// 3. Pozostałe scenariusze
	if user.TwoFactorEnabled {
		return s.prepare2FASession(ctx, user, fingerprint)
	}

	return s.finalizeLogin(ctx, user, fingerprint)
}

// region prepareEmployeeLogin
// #region prepareEmployeeLogin
func (s *authService) prepareEmployeeLogin(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	// 1. Pobieramy profil pracownika
	empProfile, err := s.employeeRepo.GetProfileByUserID(ctx, user.ID)
	if err != nil || !empProfile.Active {
		log.Error("Użytkownik ma rolę urzędniczą, ale brak aktywnego profilu", err)
		return nil, errors.ErrUnauthorized
	}

	// 2. Pobieramy aktywną kartę / poświadczenie fizyczne
	credential, err := s.employeeRepo.GetActiveCredentialByUserID(ctx, user.ID)
	if err != nil || credential.Status != model.EmployeeCredentialActive {
		log.WarnMap("Brak aktywnej karty urzędniczej dla użytkownika", map[string]any{
			"uid": user.ID,
		})
		return nil, errors.ErrInvalidCredentials
	}

	// 3. Budujemy bilet i sesję w Redis
	setupToken, sessionID, challenge, err := s.createChallengeSession(ctx, user.ID, fingerprint)
	if err != nil {
		return nil, err
	}

	// 4. Konstruujemy dane sesji z kontekstem urzędnika i kluczem z KARTY
	sessionData := redis.UserSession{
		UserID:      user.ID.String(),
		Fingerprint: crypto.HashSHA256(fingerprint),
		PublicKey:   credential.PublicKey,
		Role:        string(user.Role),
	}

	if err := s.cache.SetSetupSession(ctx, sessionID, &sessionData, 15*time.Minute); err != nil {
		log.ErrorObj("Failed to save employee setup session in Redis", err)
		return nil, errors.ErrInternal
	}

	return &http.LoginResponse{
		Type:       "employeeTrust",
		Challenge:  challenge,
		SetupToken: setupToken,
	}, nil
}

// region prepare2FASession
func (s *authService) prepare2FASession(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	// 1. Generujemy 6-cyfrowy kod (bezpiecznie)
	code, err := security.GenerateOTP(6)
	if err != nil {
		return nil, errors.ErrInternal
	}
	codeBytes := []byte(code)
	defer clear(codeBytes)

	// 2. Hashujemy kod przed zapisem (Security: At-Rest protection)
	hashedCode, err := security.HashPassword(codeBytes, nil)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 3. Tworzymy sesję 2FA (konwersja UUID na string)
	token := shared.GenerateSessionID()
	session := redis.TwoFASession{
		UserID:      user.ID.String(),
		Email:       user.Email,
		Token:       token.String(),
		CodeHash:    hashedCode,
		Fingerprint: crypto.HashSHA256(fingerprint),
		Attempts:    0,
	}

	// 4. Zapis do Redis (przekazujemy token.String() oraz wskaźnik &session)
	if err := s.cache.Set2FASession(ctx, token.String(), session, 5*time.Minute); err != nil {
		log.ErrorObj("Failed to save 2FA session in Redis", err)
		return nil, errors.ErrInternal
	}

	// 5. TODO: Wyślij kod do użytkownika
	// s.emailService.Send2FACode(user.Email, code)

	// DEBUG
	log.DebugInfo("Generated 2FA code", map[string]any{
		"email": user.Email,
		"token": token.String(),
		"code":  code,
	})

	return &http.LoginResponse{
		Type:          "2fa",
		TwoFARequired: true,
		TwoFAToken:    token.String(),
	}, nil
}

// #region finalizeLogin
func (s *authService) finalizeLogin(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	accessToken, sessionID, err := s.CreateAccessToken(ctx, user.ID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	sessionData := s.buildUserSession(user, fingerprint, "")

	if err := s.cache.SetSession(ctx, sessionID, &sessionData, s.cfg.Session.TTL); err != nil {
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(user.ID, fingerprint, nil)
	if err != nil {
		return nil, errors.ErrInternal
	}

	return &http.LoginResponse{
		TwoFARequired: false,
		AccessToken:   accessToken,
		RefreshToken:  refreshToken.Token,
		UserID:        user.ID.String(),
		ExpiresAt:     refreshToken.ExpiresAt.Unix(),
	}, nil
}

// region UpdatePassword
// #region UpdatePassword
func (s *authService) UpdatePassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return errors.ErrUserNotFound
	}

	passBytes := []byte(newPassword)
	defer clear(passBytes)

	hashed, err := security.HashPassword(passBytes, nil)
	if err != nil {
		return errors.ErrInternal
	}

	now := time.Now()
	user.Password = hashed
	user.PasswordChangedAt = &now

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	return nil
}

// region Register
// #region Register
func (s *authService) Register(username, email, rawPassword string) (*model.User, error) {
	passBytes := []byte(rawPassword)
	defer clear(passBytes)

	hash, err := security.HashPassword(passBytes, nil)
	if err != nil {
		return nil, errors.ErrInternal
	}

	now := time.Now()
	u := &model.User{
		Username:          username,
		Email:             email,
		Password:          hash,
		PasswordChangedAt: &now,
	}

	if err := s.userRepo.CreateUser(u); err != nil {
		return nil, err
	}

	return u, nil
}

// region RegisterUserDevice
// #region RegisterUserDevice
func (s *authService) RegisterUserDevice(ctx context.Context, userID uuid.UUID, fingerprint, publicKey, deviceName, platform string, isVerified bool, lastIp string) error {
	device := model.UserDevice{
		UserID: userID, DeviceFingerprint: fingerprint, PublicKey: publicKey,
		DeviceNameEncrypted: deviceName, Platform: platform, IsVerified: isVerified,
		LastIP: lastIp, IsActive: true,
	}
	return s.userRepo.SaveDevice(ctx, &device)
}

// region preparePreTrustSession
// #region preparePreTrustSession
func (s *authService) preparePreTrustSession(ctx context.Context, user *model.User, publicKey string, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	// 1. Tworzymy bilet (SetupToken) i ID sesji
	setupToken, sessionID, challenge, err := s.createChallengeSession(ctx, user.ID, fingerprint)
	if err != nil {
		return nil, err
	}

	// 2. Dane sesji dla zwykłego urządzenia
	sessionData := redis.UserSession{
		UserID:      user.ID.String(),
		Fingerprint: crypto.HashSHA256(fingerprint),
		PublicKey:   publicKey,
		Role:        string(user.Role),
	}

	// 3. Zapis w Redis
	if err := s.cache.SetSetupSession(ctx, sessionID, &sessionData, 15*time.Minute); err != nil {
		log.ErrorObj("Failed to save setup session in Redis", err)
		return nil, errors.ErrInternal
	}

	return &http.LoginResponse{
		Type:       "preTrust",
		Challenge:  challenge,
		SetupToken: setupToken,
		IsTrusted:  true,
	}, nil
}
