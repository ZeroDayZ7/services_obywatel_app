package service

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/auth-service/internal/http"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	repo "github.com/zerodayz7/platform/services/auth-service/internal/repository"
	"github.com/zerodayz7/platform/services/auth-service/internal/shared/security"
)

// AuthService definiuje pełny kontrakt biznesowy modułu autoryzacji.
// region interface
type AuthService interface {
	// Główne procesy BIZNESOWE (zostawiamy tylko to, co ma logikę)
	AttemptLogin(ctx context.Context, email string, password []byte, fingerprint string) (*http.LoginResponse, error)
	Register(username, email, rawPassword string) (*model.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, newPassword string) error
	Verify2FA(ctx context.Context, token string, code []byte, fingerprint string, ip string) (*http.Verify2FAResponse, error)
	Logout(ctx context.Context, userID uuid.UUID, sessionID string, fingerprint string) error
	RegisterDevice(ctx context.Context, userID uuid.UUID, sessionID string, clientIP string, req schemas.RegisterDeviceRequest) (*http.RegisterDeviceResponse, error)
	RefreshToken(ctx context.Context, tokenStr string, fingerprint string) (*http.RefreshResponse, error)
	VerifyDeviceSignature(ctx context.Context, userID, challenge, signature, fingerprint string) (*http.LoginResponse, error)
	// Narzędzia JWT
	CreateAccessToken(userID uuid.UUID, fingerprint string) (string, string, error)
	CreateRefreshToken(userID uuid.UUID, fingerprint string) (*model.RefreshToken, error)
	GetRefreshToken(token string) (*model.RefreshToken, error)
	RevokeRefreshToken(token string) error
	// Metody specyficzne dla logiki logowania
	CanUserLogin(user *model.User) error
}

// region struct
type authService struct {
	// Zmiana: używaj interfejsu z repozytorium
	userRepo    repo.UserRepository
	refreshRepo repo.RefreshTokenRepository
	cache       *redis.Cache
	cfg         *viper.Config
}

func NewAuthService(userRepo repo.UserRepository, refreshRepo repo.RefreshTokenRepository, cache *redis.Cache, cfg *viper.Config) AuthService {
	return &authService{
		userRepo: userRepo, refreshRepo: refreshRepo, cache: cache, cfg: cfg,
	}
}

// region VerifyDeviceSignature
func (s *authService) VerifyDeviceSignature(ctx context.Context, userIDStr, sessionID, signature, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.ErrInvalidParams
	}

	// 1. POBIERAMY CHALLENGE Z REDIS (Używając SessionID)
	cacheKey := fmt.Sprintf("auth:challenge:%s", sessionID)
	storedChallenge, err := s.cache.Get(ctx, cacheKey)
	if err != nil || storedChallenge == "" {
		log.WarnMap("Challenge not found or expired", map[string]any{"sid": sessionID})
		return nil, errors.ErrInvalidChallenge
	}

	// 1. Pobieramy urządzenie z TWOJEGO repozytorium po Fingerprincie
	device, err := s.userRepo.GetDeviceByFingerprint(ctx, userID, fingerprint)
	if err != nil || device == nil {
		log.WarnMap("Device not found or inactive", map[string]any{"user": userIDStr, "fpt": fingerprint})
		return nil, errors.ErrUntrustedDevice
	}

	log.DebugInfo("VerifyDeviceSignature - device", device)

	challengeBytes, err := base64.StdEncoding.DecodeString(storedChallenge)
	if err != nil {
		log.ErrorObj("Failed to decode challenge from Base64", err)
		return nil, errors.ErrInvalidParams
	}

	log.DebugInfo("challengeBytes", challengeBytes)

	// 2. Weryfikacja kryptograficzna Ed25519
	pubKeyBytes, err := base64.StdEncoding.DecodeString(device.PublicKey)
	if err != nil {
		return nil, errors.ErrInternal
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, errors.ErrInvalidSignature
	}

	// Kluczowy moment: sprawdzamy czy podpis pasuje do challenge'u
	if !ed25519.Verify(pubKeyBytes, challengeBytes, sigBytes) {
		log.WarnMap("SECURITY ALERT: Signature mismatch", map[string]any{"userId": userIDStr})
		return nil, errors.ErrInvalidSignature
	}

	// 3. Sukces - pobieramy dane usera i generujemy tokeny
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.ErrUserNotFound
	}

	// Używamy Twoich istniejących metod w serwisie do JWT
	accessToken, sessionID, err := s.CreateAccessToken(user.ID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(user.ID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 4. Zapisujemy sesję w Redis (używając Twojego s.cache)
	// Pobieramy role - na razie "USER", docelowo z bazy
	roles := []string{"USER"}
	err = s.cache.SetSession(ctx, sessionID, redis.UserSession{
		UserID:      user.ID.String(),
		Fingerprint: fingerprint,
		Roles:       roles,
	}, s.cfg.Session.TTL)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 5. Zwracamy odpowiedź zgodną z Twoim http.LoginResponse
	return &http.LoginResponse{
		Type:         "fullSuccess",
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		UserID:       fmt.Sprint(user.ID),
		// Tu nie dodajemy UserResponse, jeśli go nie masz w strukturze http
	}, nil
}

// region RefreshToken
func (s *authService) RefreshToken(ctx context.Context, tokenStr string, fingerprint string) (*http.RefreshResponse, error) {
	log := shared.GetLogger()

	// 1. Pobranie i walidacja Refresh Tokena z bazy
	rt, err := s.refreshRepo.GetByToken(tokenStr)
	if err != nil || rt.Revoked || rt.ExpiresAt.Before(time.Now()) {
		log.WarnObj("Invalid, revoked or expired refresh token", tokenStr)
		return nil, errors.ErrInvalidToken
	}

	// 2. Weryfikacja Fingerprint (Security Binding)
	if rt.DeviceFingerprint != fingerprint {
		log.WarnMap("SECURITY ALERT: Refresh token used on different device!", map[string]any{
			"user_id":      rt.UserID,
			"expected_fpt": rt.DeviceFingerprint,
			"received_fpt": fingerprint,
		})
		// Opcjonalnie: s.RevokeAllUserTokens(rt.UserID)
		return nil, errors.ErrInvalidToken
	}

	// 3. Generowanie nowych poświadczeń
	// Tworzymy nowy Access Token i nowe SessionID (SID)
	accessToken, newSessionID, err := s.CreateAccessToken(rt.UserID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 2. Pobierz aktualne dane użytkownika z bazy
	user, err := s.userRepo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 3. Pobierz role (np. z obiektu user lub dedykowanej tabeli)
	roles := []string{"USER"}
	// if user.IsAdmin {
	// 	roles = append(roles, "ADMIN")
	// }

	// 4. Aktualizacja sesji w Redis z ROLAMI
	err = s.cache.SetSession(ctx, newSessionID, redis.UserSession{
		UserID:      user.ID.String(),
		Fingerprint: fingerprint,
		Roles:       roles,
	}, s.cfg.Session.TTL)
	if err != nil {
		log.ErrorObj("Failed to save session in Redis", err)
		return nil, errors.ErrInternal
	}

	return &http.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: rt.Token, // Zwracamy ten sam lub generujemy nowy (Rotation)
		UserID:       rt.UserID.String(),
		Roles:        roles,
		ExpiresAt:    time.Now().Add(s.cfg.JWT.AccessTTL).Unix(),
	}, nil
}

// region RegisterDevice
func (s *authService) RegisterDevice(ctx context.Context, userID uuid.UUID, sessionID string, clientIP string, req schemas.RegisterDeviceRequest) (*http.RegisterDeviceResponse, error) {
	log := shared.GetLogger()

	// 1. WERYFIKACJA KRYPTOGRAFICZNA
	storedChallenge, err := s.cache.GetChallenge(ctx, sessionID)
	if err != nil {
		log.WarnMap("Challenge expired or not found", map[string]any{
			"user_id": userID,
			"sid":     sessionID,
		})
		return nil, errors.ErrSessionExpired
	}

	// Dekodujemy klucz publiczny
	pubKeyBytes, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
		return nil, errors.ErrInvalidPairingData
	}

	// Dekodujemy sygnaturę
	sigBytes, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		return nil, errors.ErrInvalidPairingData
	}

	// Weryfikacja podpisu
	if !ed25519.Verify(pubKeyBytes, []byte(storedChallenge), sigBytes) {
		log.ErrorMap("Kryptograficzna weryfikacja urządzenia nieudana", map[string]any{"user_id": userID})
		return nil, errors.ErrVerificationFailed
	}

	err = s.userRepo.SaveDevice(ctx, &model.UserDevice{
		UserID:              userID,
		DeviceFingerprint:   req.DeviceFingerprint,
		PublicKey:           req.PublicKey,
		DeviceNameEncrypted: req.DeviceNameEncrypted,
		Platform:            req.Platform,
		IsVerified:          true,
		IsActive:            true,
		LastIp:              clientIP,
	})
	if err != nil {
		log.ErrorObj("Failed to save device", err)
		return nil, errors.ErrInternal
	}

	// 3. SYNCHRONIZACJA (Logika biznesowa - Zamiana Setup na Full Session)
	if sessionID != "" {
		// Pobieramy sesję z prefixu SETUP (bo tam trafiła po 2FA)
		setupSess, sessErr := s.cache.GetSetupSession(ctx, sessionID)
		if sessErr != nil || setupSess == nil {
			log.WarnMap("Setup session not found or expired", map[string]any{"sid": sessionID})
			return nil, errors.ErrSessionExpired
		}

		// Weryfikacja czy Fingerprint urządzenia zgadza się z tym z etapu logowania
		if setupSess.Fingerprint != req.DeviceFingerprint {
			log.WarnMap("Fingerprint mismatch during registration", map[string]any{
				"expected": setupSess.Fingerprint,
				"received": req.DeviceFingerprint,
			})
			return nil, errors.ErrInvalidDeviceFingerprint
		}

		// Usuwamy sesję tymczasową, bo za chwilę stworzymy pełną sesję (newSID)
		_ = s.cache.DeleteChallenge(ctx, sessionID)
		_ = s.cache.DeleteSetupSession(ctx, sessionID)
		log.DebugInfo("Setup session cleared, upgrading to full session", sessionID)
	}
	// 4. GENEROWANIE NOWYCH POŚWIADCZEŃ
	accessToken, newSID, err := s.CreateAccessToken(userID, req.DeviceFingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(userID, req.DeviceFingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 1. Pobierz pełne dane użytkownika (w tym role/rbac)
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 2. Przygotuj dane do sesji (np. role jako string slice)
	roles := []string{"USER"} // Tu pobierz role z obiektu user, np. user.Roles

	// 3. Zapisz BOGATĄ sesję w cache (używając struktury UserSession)
	sessionData := redis.UserSession{
		UserID:      user.ID.String(),
		Fingerprint: req.DeviceFingerprint,
		Roles:       roles,
	}

	if err = s.cache.SetSession(ctx, newSID, sessionData, s.cfg.Session.TTL); err != nil {
		log.ErrorObj("Failed to save session", err)
		return nil, errors.ErrInternal
	}
	// 5. FINALIZACJA

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
			Roles:       []string{"USER"},
		},
	}, nil
}

// region Logout
func (s *authService) Logout(ctx context.Context, userID uuid.UUID, sessionID string, fingerprint string) error {
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
func (s *authService) Verify2FA(ctx context.Context, token string, code []byte, fingerprint string, ip string) (*http.Verify2FAResponse, error) {
	log := shared.GetLogger()
	// 1. Pobieranie sesji 2FA z Cache
	session, err := s.cache.Get2FASession(ctx, token)
	if err != nil {
		return nil, errors.ErrInvalidCredentials
	}
	codeStr := strings.TrimSpace(string(code))
	log.DebugInfo("2FA compare", map[string]any{
		"plain": codeStr,
		"hash":  session.CodeHash,
	})
	// 2. Weryfikacja kodu bcryptem
	valid, err := security.VerifyPassword(code, session.CodeHash)

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

	// 5. Generowanie tokenów i sesji głównej
	setupToken, sessionID, err := s.CreateAccessToken(uid, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	err = s.cache.SetSetupSession(ctx, sessionID, redis.UserSession{
		UserID:      session.UserID,
		Fingerprint: fingerprint,
	}, s.cfg.Session.TTL)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 6. Generowanie Challenge (Ed25519)
	challenge, err := shared.GenerateRandomChallenge(32)
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
	log.DebugJSON("[DEBUG] Sending 2FA Response:",
		response,
	)

	// Opcjonalnie: zrzut do JSONa w logach, żeby sprawdzić klucze
	// responseJson, _ := json.Marshal(response)
	// log.Printf("[DEBUG] Full JSON: %s", string(responseJson))

	return response, nil
}

// region AttemptLogin
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

	valid, err := security.VerifyPassword(password, user.Password)
	if err != nil || !valid {
		_ = s.userRepo.IncrementUserFailedLogin(user.ID)
		return nil, errors.ErrInvalidCredentials
	}

	if user.FailedLoginAttempts > 0 {
		_ = s.userRepo.ResetFailedLoginAttempts(user.ID)
	}

	device, err := s.userRepo.GetDeviceByFingerprint(ctx, user.ID, fingerprint)
	log.DebugDB("SCENARIUSZ A", device)

	// SCENARIUSZ A: Urządzenie jest znane i zweryfikowane
	if err == nil && device != nil && device.IsVerified && device.IsActive {

		// 1. Najpierw bilet (SetupToken) i unikalne ID sesji (v7)
		setupToken, sessionID, err := s.CreateSetupToken(user.ID, fingerprint)
		if err != nil {
			log.ErrorObj("Failed to create setup token", err)
			return nil, errors.ErrInternal
		}

		// 2. Generujemy challenge
		challenge, err := shared.GenerateRandomChallenge(32)
		if err != nil {
			log.ErrorObj("Failed to generate challenge", err)
			return nil, errors.ErrInternal
		}

		// 3. Zapisujemy w Redis pod kluczem SESJI (najbezpieczniej)
		if err := s.cache.SetChallenge(ctx, sessionID, challenge, 5*time.Minute); err != nil {
			log.ErrorObj("Failed to save challenge in Redis", err)
			return nil, errors.ErrInternal
		}

		log.DebugInfo("Pre-trust session prepared", map[string]any{
			"uid": user.ID,
			"sid": sessionID,
		})

		// 4. ZWRACAMY dane
		return &http.LoginResponse{
			Type:       "preTrust",
			Challenge:  challenge,
			SetupToken: setupToken,
			IsTrusted:  true,
		}, nil
	}

	if user.TwoFactorEnabled {
		return s.prepare2FASession(ctx, user, fingerprint)
	}

	return s.finalizeLogin(ctx, user, fingerprint)
}

// region prepare2FASession
func (s *authService) prepare2FASession(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()
	// 1. Generujemy 6-cyfrowy kod (bezpiecznie)
	code, err := shared.GenerateSecureOTP()
	if err != nil {
		return nil, errors.ErrInternal
	}
	// 2. Hashujemy kod przed zapisem (Security: At-Rest protection)
	hashedCode, err := security.HashPassword(code) // Używamy bcrypt, który już masz
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 3. Tworzymy sesję 2FA
	token := shared.GenerateSessionID()
	session := redis.TwoFASession{
		UserID:      user.ID.String(),
		Email:       user.Email,
		Token:       token,
		CodeHash:    hashedCode,
		Fingerprint: fingerprint,
		Attempts:    0,
	}

	// 4. Zapis do Redis (Metoda sama robi Marshal i dodaje prefix klucza)
	if err := s.cache.Set2FASession(ctx, token, session, 5*time.Minute); err != nil {
		log.ErrorObj("Failed to save 2FA session in Redis", err)
		return nil, errors.ErrInternal
	}

	// 5. TODO: Wyślij kod do użytkownika
	// s.emailService.Send2FACode(user.Email, code)

	// DEBUG: W fazie deweloperskiej wypisz kod w konsoli
	log.DebugInfo("Generated 2FA code", map[string]any{
		"email": user.Email,
		"token": token,
		"code":  code,
	})

	return &http.LoginResponse{
		Type:          "2fa",
		TwoFARequired: true,
		TwoFAToken:    token,
	}, nil
}

// region finalizeLogin
func (s *authService) finalizeLogin(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	accessToken, sessionID, err := s.CreateAccessToken(user.ID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	err = s.cache.SetSession(ctx, sessionID, redis.UserSession{
		UserID:      user.ID.String(),
		Fingerprint: fingerprint,
		Roles:       nil,
	}, s.cfg.Session.TTL)
	if err != nil {
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(user.ID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	return &http.LoginResponse{
		TwoFARequired: false,
		AccessToken:   accessToken,
		RefreshToken:  refreshToken.Token,
		UserID:        fmt.Sprint(user.ID),
		ExpiresAt:     refreshToken.ExpiresAt.Unix(),
	}, nil
}

// region UpdatePassword
func (s *authService) UpdatePassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return errors.ErrUserNotFound
	}

	hashed, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashed
	return s.userRepo.Update(ctx, user)
}

// region CreateAccessToken
func (s *authService) CreateAccessToken(userID uuid.UUID, fingerprint string) (string, string, error) {
	sessionID := shared.GenerateSessionID()
	claims := jwt.MapClaims{
		"uid": userID,
		"sid": sessionID,
		"fpt": fingerprint,
	}

	token, err := security.GenerateJWT(claims, s.cfg.JWT.AccessSecret, s.cfg.JWT.AccessTTL)
	return token, sessionID, err
}

// region CreateSetupToken
func (s *authService) CreateSetupToken(userID uuid.UUID, fingerprint string) (string, string, error) {
	sessionID := shared.GenerateSessionID()
	claims := jwt.MapClaims{
		"uid":   userID.String(),
		"sid":   sessionID,
		"fpt":   fingerprint,
		"scope": "device_verify",
	}

	token, err := security.GenerateJWT(
		claims,
		s.cfg.JWT.AccessSecret,
		15*time.Minute,
	)

	return token, sessionID, err
}

// region CreateRefreshToken
func (s *authService) CreateRefreshToken(userID uuid.UUID, fingerprint string) (*model.RefreshToken, error) {
	rawToken, _ := security.GenerateRefreshToken()
	hash := sha256.Sum256([]byte(rawToken))
	hashedTokenHex := hex.EncodeToString(hash[:])

	rt := &model.RefreshToken{
		UserID:            userID,
		Token:             hashedTokenHex,
		DeviceFingerprint: fingerprint,
		ExpiresAt:         time.Now().Add(s.cfg.JWT.RefreshTTL),
	}

	if err := s.refreshRepo.Save(rt); err != nil {
		return nil, err
	}
	rt.Token = rawToken
	return rt, nil
}

// region GetRefreshToken
func (s *authService) GetRefreshToken(token string) (*model.RefreshToken, error) {
	return s.refreshRepo.GetByToken(token)
}

// region RevokeRefreshToken
func (s *authService) RevokeRefreshToken(token string) error {
	rt, err := s.refreshRepo.GetByToken(token)
	if err != nil {
		return err
	}
	rt.Revoked = true
	return s.refreshRepo.Update(rt)
}

// region Register
func (s *authService) Register(username, email, rawPassword string) (*model.User, error) {
	hash, _ := security.HashPassword(rawPassword)
	u := &model.User{Username: username, Email: email, Password: hash}
	err := s.userRepo.CreateUser(u)
	return u, err
}

// region RegisterUserDevice
func (s *authService) RegisterUserDevice(ctx context.Context, userID uuid.UUID, fingerprint, publicKey, deviceName, platform string, isVerified bool, lastIp string) error {
	device := model.UserDevice{
		UserID: userID, DeviceFingerprint: fingerprint, PublicKey: publicKey,
		DeviceNameEncrypted: deviceName, Platform: platform, IsVerified: isVerified,
		LastIp: lastIp, IsActive: true,
	}
	return s.userRepo.SaveDevice(ctx, &device)
}

// region CanUserLogin
func (s *authService) CanUserLogin(user *model.User) error {
	switch user.Status {
	case model.StatusActive:
		return nil
	case model.StatusSuspended:
		return errors.ErrAccountSuspended
	case model.StatusBanned:
		return errors.ErrAccountBanned
	default:
		return errors.ErrInternal
	}
}
