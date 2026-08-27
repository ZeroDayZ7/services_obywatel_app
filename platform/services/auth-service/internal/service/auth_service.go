package service

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/kms"
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
	Verify2FA(ctx context.Context, token uuid.UUID, code []byte, fingerprint string, ip string) (*http.LoginResponse, error)
	Resend2FACode(ctx context.Context, email string, token uuid.UUID) error
	Logout(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, fingerprint string) error
	RegisterDevice(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, clientIP string, req schemas.RegisterDeviceRequest) (*http.RegisterDeviceResponse, error)
	CreateTemporarySession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, clientIP string) (*http.LoginResponse, error)
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

// region AttemptLogin
func (s *authService) AttemptLogin(ctx context.Context, email string, password []byte, fingerprint string) (*http.LoginResponse, error) {
	defer kms.ZeroBytes(password)

	log := shared.GetLogger()

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		// Ochrona przed timing attack
		security.VerifyPasswordDummy(password, nil)
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

// #region AttemptLoginStep2
func (s *authService) AttemptLoginStep2(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, signature string, fingerprint string, clientIP string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	// 1. Pobranie aktywnego poświadczenia pracownika (klucz publiczny z DB)
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

	pubKeyBytes, err := hex.DecodeString(cred.PublicKey)
	if err != nil {
		log.WarnMap("[AttemptLoginStep2] Nieprawidłowy format klucza publicznego w DB", map[string]any{
			"user_id": userID,
			"err":     err,
		})
		return nil, errors.ErrInvalidSignature
	}

	// 2. Weryfikacja challenge, podpisu i usunięcie z Redisa w jednej funkcji
	if err := s.verifyChallengeSession(ctx, sessionID, signature, pubKeyBytes); err != nil {
		log.WarnMap("SECURITY ALERT: Nieprawidłowy podpis pracownika lub wygasłe wyzwanie", map[string]any{
			"user_id": userID,
			"card_sn": cred.CardSerialNumber,
			"err":     err,
		})
		return nil, errors.ErrInvalidSignature
	}

	// 3. Pobranie użytkownika
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

	// 4. Metadane profilu pracownika - przypisujemy pobrany profil do struktury usera
	if user.Role != model.RoleCitizen && user.Role != model.RoleUser {
		empProfile, err := s.employeeRepo.GetProfileByUserID(ctx, user.ID)
		if err == nil && empProfile != nil {
			user.EmployeeProfile = empProfile
		}
	}

	// 5. Generowanie tokenów i sesji
	accessToken, newSessionID, err := s.CreateAccessToken(ctx, user.ID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(user.ID, fingerprint, nil)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// Korzystamy z ujednoliconej budowy sesji
	sessionData := s.buildUserSession(user, fingerprint, cred.PublicKey, false)

	if err := s.cache.SetSession(ctx, newSessionID, &sessionData, s.cfg.Session.TTL); err != nil {
		return nil, errors.ErrInternal
	}

	expiresAt := time.Now().Add(s.cfg.JWT.AccessTTL).Unix()

	return &http.LoginResponse{
		Type: http.LoginResultSuccess,
		Success: &http.LoginSuccessData{
			AccessToken:  accessToken,
			RefreshToken: refreshToken.Token,
			UserID:       user.ID.String(),
			ExpiresAt:    expiresAt,
		},
	}, nil
}

// #region Verify2FA
func (s *authService) Verify2FA(ctx context.Context, token uuid.UUID, code []byte, fingerprint string, ip string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	// 1. Czyszczenie wrażliwego bufora z kodem OTP w pamięci RAM
	defer kms.ZeroBytes(code)

	// 2. Pobieranie sesji 2FA z Cache (Redis)
	session, err := s.cache.Get2FASession(ctx, token)
	if err != nil {
		return nil, errors.ErrInvalidCredentials
	}

	// 3. Szybka weryfikacja SHA-256 (Constant Time Compare chroniący przed Timing Attack)
	inputHash := security.HashOTP(code)
	if subtle.ConstantTimeCompare([]byte(inputHash), []byte(session.CodeHash)) != 1 {
		status, _ := s.cache.Verify2FAAttempt(ctx, token, 5, 5*time.Minute)
		log.DebugInfo("2FA verification failed", map[string]any{
			"status": status,
			"token":  token.String(),
		})

		if status == "locked" {
			return nil, errors.Err2FALocked
		}
		return nil, errors.ErrInvalid2FACode
	}

	// 4. Jednorazowe użycie kodu — czyszczenie sesji 2FA z Redisa
	_ = s.cache.Delete2FASession(ctx, token)

	// 5. Pobieranie użytkownika i aktualizacja metadanych
	uid, err := uuid.Parse(session.UserID)
	if err != nil {
		return nil, errors.ErrInternal
	}

	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil || user == nil {
		return nil, errors.ErrUserNotFound
	}

	user.LastLogin = time.Now()
	user.LastIP = ip
	_ = s.userRepo.Update(ctx, user)

	// 6. Tworzenie wyzwania (Challenge) dla kolejnego kroku (PreTrust / Rejestracja urządzenia)
	setupToken, sessionID, err := s.CreateSetupToken(ctx, uid, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	challenge, err := security.GenerateRandomString(32)
	if err != nil {
		log.ErrorObj("Failed to generate secure challenge", err)
		return nil, errors.ErrInternal
	}

	// Zapisujemy sesję Setup/Challenge w Redis ściśle na 5 minut
	err = s.cache.SetSetupSession(ctx, sessionID, &redis.SetupSession{
		UserID:      session.UserID,
		Challenge:   challenge,
		Fingerprint: fingerprint,
	}, 5*time.Minute)
	if err != nil {
		return nil, errors.ErrInternal
	}

	return &http.LoginResponse{
		Type: http.LoginResultPreTrust,
		PreTrust: &http.LoginPreTrustData{
			SetupToken: setupToken,
			Challenge:  challenge,
		},
	}, nil
}

// #region Resend2FACode
func (s *authService) Resend2FACode(ctx context.Context, email string, token uuid.UUID) error {
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

	codeBytes, err := security.GenerateOTPBytes(6)
	if err != nil {
		return errors.ErrInternal
	}
	defer kms.ZeroBytes(codeBytes)

	session.CodeHash = security.HashOTP(codeBytes)

	// 6. Odświeżamy sesję 2FA w Redis (z resetem TTL na kolejne 5 minut)
	if err := s.cache.Set2FASession(ctx, token, *session, 5*time.Minute); err != nil {
		log.ErrorObj("Resend2FA: failed to update 2FA session in Redis", err)
		return errors.ErrInternal
	}

	// Używamy metody z wstrzykniętej instancji s.cfg
	if s.cfg.IsLocalDev() {
		log.DebugInfo("Resent 2FA code successfully", map[string]any{
			"email": email,
			"token": token,
			"code":  string(codeBytes),
		})
	}

	// TODO: Wyślij kod do użytkownika (SMS/Email)
	// s.emailService.Send2FACode(user.Email, codeBytes)

	return nil
}

// region prepareEmployeeLogin
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
	sessionData := redis.SetupSession{
		UserID:      user.ID.String(),
		Fingerprint: fingerprint,
		PublicKey:   credential.PublicKey,
		Role:        string(user.Role),
	}

	if err := s.cache.SetSetupSession(ctx, sessionID, &sessionData, 15*time.Minute); err != nil {
		log.ErrorObj("Failed to save employee setup session in Redis", err)
		return nil, errors.ErrInternal
	}

	return &http.LoginResponse{
		Type: http.LoginResultEmployeeTrust,
		EmployeeTrust: &http.LoginEmployeeTrustData{
			Challenge:  challenge,
			SetupToken: setupToken,
		},
	}, nil
}

// #region prepare2FASession
func (s *authService) prepare2FASession(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	codeBytes, err := security.GenerateOTPBytes(6)
	if err != nil {
		return nil, errors.ErrInternal
	}
	defer kms.ZeroBytes(codeBytes)

	hashedCode := security.HashOTP(codeBytes)
	token := shared.GenerateSessionID()

	session := redis.TwoFASession{
		UserID:      user.ID.String(),
		Token:       token.String(),
		CodeHash:    hashedCode,
		Fingerprint: fingerprint,
		Attempts:    0,
	}

	if err := s.cache.Set2FASession(ctx, token, session, 5*time.Minute); err != nil {
		log.ErrorObj("Failed to save 2FA session in Redis", err)
		return nil, errors.ErrInternal
	}

	// 5. TODO: Wyślij kod do użytkownika (SMS/Email)
	// s.emailService.Send2FACode(user.Email, codeBytes)

	// Używamy metody z wstrzykniętej instancji s.cfg
	if s.cfg.IsLocalDev() {
		log.DebugInfo("Generated 2FA code", map[string]any{
			"email": user.Email,
			"token": token.String(),
			"code":  string(codeBytes),
		})
	}

	return &http.LoginResponse{
		Type: http.LoginResult2FARequired,
		TwoFA: &http.Login2FAData{
			TwoFARequired: true,
			TwoFAToken:    token.String(),
		},
	}, nil
}

// region finalizeLogin
//
// #region finalizeLogin
func (s *authService) finalizeLogin(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	// 1. Generujemy krótkotrwały Access Token (15 min / Read-Only w sesji dla niezaufanego urządzenia)
	accessToken, sessionID, err := s.CreateAccessToken(ctx, user.ID, fingerprint)
	if err != nil {
		log.ErrorObj("Failed to create access token during login finalization", err)
		return nil, errors.ErrInternal
	}

	// 2. Budujemy dane sesji z flagą ReadOnly/Krótkim czasem życia
	sessionData := s.buildUserSession(user, fingerprint, "", true)
	ttl := s.cfg.Session.TTL

	if err := s.cache.SetSession(ctx, sessionID, &sessionData, ttl); err != nil {
		log.ErrorObj("Failed to save session in Redis during login finalization", err)
		return nil, errors.ErrInternal
	}

	// 3. Dla niezaufanego urządzenia celowo NIE generujemy Refresh Tokena.
	// Wymusza to ponowną autoryzację po upływie 15 minut lub parowanie urządzenia.

	expiresAt := time.Now().Add(ttl).Unix()

	return &http.LoginResponse{
		Type: http.LoginResultSuccess,
		Success: &http.LoginSuccessData{
			AccessToken:  accessToken,
			RefreshToken: "",
			UserID:       user.ID.String(),
			ExpiresAt:    expiresAt,
		},
	}, nil
}

// region preparePreTrustSession
// #region preparePreTrustSession
// #region preparePreTrustSession
func (s *authService) preparePreTrustSession(ctx context.Context, user *model.User, publicKey string, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	// 1. Tworzymy bilet (SetupToken) i ID sesji
	setupToken, sessionID, challenge, err := s.createChallengeSession(ctx, user.ID, fingerprint)
	if err != nil {
		return nil, err
	}

	// 2. Dane sesji dla zaufanego urządzenia
	sessionData := redis.SetupSession{
		UserID:      user.ID.String(),
		Fingerprint: fingerprint,
		PublicKey:   publicKey,
		Role:        string(user.Role),
	}

	// 3. Zapis w Redis (sesja wyzwania ważna 15 minut)
	if err := s.cache.SetSetupSession(ctx, sessionID, &sessionData, 15*time.Minute); err != nil {
		log.ErrorObj("Failed to save setup session in Redis", err)
		return nil, errors.ErrInternal
	}

	return &http.LoginResponse{
		Type: http.LoginResultPreTrust,
		PreTrust: &http.LoginPreTrustData{
			SetupToken: setupToken,
			Challenge:  challenge,
		},
	}, nil
}

// #region CreateTemporarySession
func (s *authService) CreateTemporarySession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, clientIP string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	// 1. Pobieramy sesję setup/challenge z Redisa
	setupSession, err := s.cache.GetSetupSession(ctx, sessionID)
	if err != nil || setupSession == nil {
		log.WarnMap("[CreateTemporarySession] Brak lub wygasła sesja setupToken", map[string]any{
			"user_id":    userID,
			"session_id": sessionID,
		})
		return nil, errors.ErrUnauthorized
	}

	// 2. Pobieramy użytkownika z bazy danych
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		log.WarnMap("[CreateTemporarySession] Użytkownik nie istnieje", map[string]any{
			"user_id": userID,
		})
		return nil, errors.ErrUserNotFound
	}

	if err := s.CanUserLogin(user); err != nil {
		return nil, err
	}

	// 3. Usuwamy zużytą sesję setupToken z Redisa
	_ = s.cache.DeleteSetupSession(ctx, sessionID)

	// 4. Generujemy nowy Access Token oraz Refresh Token dla sesji tymczasowej
	accessToken, newSessionID, err := s.CreateAccessToken(ctx, user.ID, setupSession.Fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 5. Budujemy pełną sesję użytkownika (z flaga readOnly = false, aby umożliwić standardowe działanie)
	sessionData := s.buildUserSession(user, setupSession.Fingerprint, "", false)

	if err := s.cache.SetSession(ctx, newSessionID, &sessionData, s.cfg.Session.TTL); err != nil {
		log.ErrorObj("[CreateTemporarySession] Błąd zapisu sesji w Redis", err)
		return nil, errors.ErrInternal
	}

	// 6. Aktualizacja metadanych logowania
	user.LastLogin = time.Now()
	user.LastIP = clientIP
	_ = s.userRepo.Update(ctx, user)

	expiresAt := time.Now().Add(s.cfg.JWT.AccessTTL).Unix()

	return &http.LoginResponse{
		Type: http.LoginResultTemporarySuccess,
		Success: &http.LoginSuccessData{
			AccessToken: accessToken,
			ExpiresAt:   expiresAt,
		},
	}, nil
}

// #endregion
