package service

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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
type AuthService interface {
	// Główne procesy BIZNESOWE (zostawiamy tylko to, co ma logikę)
	AttemptLogin(ctx context.Context, email string, password []byte, fingerprint string) (*http.LoginResponse, error)
	Register(username, email, rawPassword string) (*model.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, newPassword string) error
	Verify2FA(ctx context.Context, token string, code []byte, fingerprint string, ip string) (*http.Verify2FAResponse, error)
	Logout(ctx context.Context, refreshToken, userID, sessionID string) error
	RegisterDevice(ctx context.Context, userID uuid.UUID, sessionID string, clientIP string, req schemas.RegisterDeviceRequest) (*http.RegisterDeviceResponse, error)
	RefreshToken(ctx context.Context, tokenStr string, fingerprint string) (*http.RefreshResponse, error)

	// Narzędzia JWT
	CreateAccessToken(userID uuid.UUID, fingerprint string) (string, string, error)
	CreateRefreshToken(userID uuid.UUID, fingerprint string) (*model.RefreshToken, error)
	GetRefreshToken(token string) (*model.RefreshToken, error)
	RevokeRefreshToken(token string) error
	UpdateRefreshTokensFingerprint(userID uuid.UUID, oldFP, newFP string) error

	// Metody specyficzne dla logiki logowania
	CanUserLogin(user *model.User) error
}

type authService struct {
	// Zmiana: używaj interfejsu z repozytorium
	userRepo    repo.UserRepository
	refreshRepo repo.RefreshTokenRepository
	cache       *redis.Cache
	cfg         *viper.Config
}

func NewAuthService(
	userRepo repo.UserRepository,
	refreshRepo repo.RefreshTokenRepository,
	cache *redis.Cache,
	cfg *viper.Config,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		cache:       cache,
		cfg:         cfg,
	}
}

// --- Implementacja Metod ---
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
	accessToken, sessionID, err := s.CreateAccessToken(rt.UserID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 4. Aktualizacja sesji w Redis
	// Bardzo ważne: Stara sesja wygaśnie, nowa sesja (z nowym SID) zostaje zarejestrowana
	err = s.cache.SetSession(ctx, sessionID, rt.UserID.String(), fingerprint, s.cfg.Session.TTL)
	if err != nil {
		log.ErrorObj("Failed to save session in Redis", err)
		return nil, errors.ErrInternal
	}

	return &http.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: rt.Token, // Zwracamy ten sam lub generujemy nowy (Rotation)
		UserID:       rt.UserID.String(),
		ExpiresAt:    time.Now().Add(s.cfg.JWT.AccessTTL).Unix(),
	}, nil
}

func (s *authService) RegisterDevice(ctx context.Context, userID uuid.UUID, sessionID string, clientIP string, req schemas.RegisterDeviceRequest) (*http.RegisterDeviceResponse, error) {
	log := shared.GetLogger()

	// 1. WERYFIKACJA KRYPTOGRAFICZNA
	challengeKey := fmt.Sprintf("challenge:%s", userID)
	storedChallenge, err := s.cache.Get(ctx, challengeKey)
	if err != nil {
		log.WarnMap("Challenge expired or not found", map[string]any{"user_id": userID})
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

	// Usuwamy challenge
	_ = s.cache.Del(ctx, challengeKey)

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

	// 3. SYNCHRONIZACJA (Logika biznesowa)
	if sessionID != "" {
		currentSession, sessErr := s.cache.GetSession(ctx, sessionID)
		if sessErr == nil && currentSession != nil && currentSession.Fingerprint != req.DeviceFingerprint {
			log.InfoMap("Syncing device fingerprint in background", map[string]any{"sid": sessionID})
			_ = s.UpdateRefreshTokensFingerprint(userID, currentSession.Fingerprint, req.DeviceFingerprint)
			_ = s.cache.UpdateSessionFingerprint(ctx, sessionID, req.DeviceFingerprint)
		}
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

	// Rejestracja nowej sesji w cache
	if err = s.cache.SetSession(ctx, newSID, userID.String(), req.DeviceFingerprint, s.cfg.Session.TTL); err != nil {
		log.ErrorObj("Failed to save session", err)
		return nil, errors.ErrInternal
	}

	// 5. FINALIZACJA
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.ErrInternal
	}

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

func (s *authService) Logout(ctx context.Context, refreshToken, userID, sessionID string) error {
	// 1. Usuń refresh token z DB (Revoke)
	if err := s.RevokeRefreshToken(refreshToken); err != nil {
		// Logujemy błąd wewnętrzny, ale nie przerywamy, jeśli sesja w Redis jeszcze istnieje
		shared.GetLogger().ErrorObj("Failed to revoke refresh token in DB", err)
	}

	// 2. Pobierz userID z Redis po sessionID i sprawdź zgodność
	storedUserID, err := s.cache.GetUserIDBySession(ctx, sessionID)
	if err != nil {
		return errors.ErrUnauthorized // Sesja już nie istnieje lub błąd Redis
	}

	if storedUserID != userID {
		shared.GetLogger().WarnMap("Logout session mismatch", map[string]any{
			"provided": userID,
			"stored":   storedUserID,
		})
		return errors.ErrUnauthorized // Próba wylogowania cudzej sesji
	}

	// 3. Usuń sesję z Redis
	if err := s.cache.DeleteSession(ctx, sessionID); err != nil {
		return errors.ErrInternal
	}

	return nil
}

func (s *authService) Verify2FA(ctx context.Context, token string, code []byte, fingerprint string, ip string) (*http.Verify2FAResponse, error) {
	log := shared.GetLogger()
	// 1. Pobieranie sesji 2FA z Cache
	key := fmt.Sprintf("login:2fa:%s", token)
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, errors.ErrInvalidCredentials
	}
	log.DebugInfo("Body", key)
	var session redis.TwoFASession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, errors.ErrInternal
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
		status, _ := s.cache.Verify2FAAttempt(ctx, key, 5, 5*time.Minute)
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
	_ = s.cache.Del(ctx, key)

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

	err = s.cache.SetSession(ctx, sessionID, session.UserID, fingerprint, s.cfg.Session.TTL)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 6. Generowanie Challenge (Ed25519)
	challenge := shared.GenerateUuid()
	if err := s.cache.Set(ctx, fmt.Sprintf("challenge:%s", uid), challenge, 5*time.Minute); err != nil {
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

func (s *authService) AttemptLogin(ctx context.Context, email string, password []byte, fingerprint string) (*http.LoginResponse, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.ErrInvalidCredentials
	}

	if err := s.CanUserLogin(user); err != nil {
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

	if user.TwoFactorEnabled {
		return s.prepare2FASession(ctx, user, fingerprint)
	}

	return s.finalizeLogin(ctx, user, fingerprint)
}

// prepare2FASession - Naprawiono błędy unusedparams używając _ lub logiki
func (s *authService) prepare2FASession(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()
	// 1. Generujemy 6-cyfrowy kod (bezpiecznie)
	code := fmt.Sprintf("%06d", shared.RandInt(100000, 999999))

	// 2. Hashujemy kod przed zapisem (Security: At-Rest protection)
	hashedCode, err := security.HashPassword(code) // Używamy bcrypt, który już masz
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 3. Tworzymy sesję 2FA
	token := shared.GenerateUuid()
	session := redis.TwoFASession{
		UserID:      user.ID.String(),
		Email:       user.Email,
		Token:       token,
		CodeHash:    hashedCode,
		Fingerprint: fingerprint,
		Attempts:    0,
	}

	// 4. Zapis do Redis (np. na 5 minut)
	data, _ := json.Marshal(session)
	key := fmt.Sprintf("login:2fa:%s", token)
	if err := s.cache.Set(ctx, key, data, 5*time.Minute); err != nil {
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
		TwoFARequired: true,
		TwoFAToken:    token,
	}, nil
}

func (s *authService) finalizeLogin(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	accessToken, sessionID, err := s.CreateAccessToken(user.ID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	err = s.cache.SetSession(ctx, sessionID, fmt.Sprint(user.ID), fingerprint, s.cfg.Session.TTL)
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

func (s *authService) CreateAccessToken(userID uuid.UUID, fingerprint string) (string, string, error) {
	sessionID := shared.GenerateUuid()
	claims := jwt.MapClaims{
		"uid": userID,
		"sid": sessionID,
		"fpt": fingerprint,
	}

	token, err := security.GenerateJWT(claims, s.cfg.JWT.AccessSecret, s.cfg.JWT.AccessTTL)
	return token, sessionID, err
}

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

func (s *authService) GetRefreshToken(token string) (*model.RefreshToken, error) {
	return s.refreshRepo.GetByToken(token)
}

func (s *authService) RevokeRefreshToken(token string) error {
	rt, err := s.refreshRepo.GetByToken(token)
	if err != nil {
		return err
	}
	rt.Revoked = true
	return s.refreshRepo.Update(rt)
}

func (s *authService) UpdateRefreshTokensFingerprint(userID uuid.UUID, oldFP, newFP string) error {
	return s.refreshRepo.UpdateFingerprintByUser(userID, oldFP, newFP)
}

func (s *authService) Register(username, email, rawPassword string) (*model.User, error) {
	hash, _ := security.HashPassword(rawPassword)
	u := &model.User{Username: username, Email: email, Password: hash}
	err := s.userRepo.CreateUser(u)
	return u, err
}

func (s *authService) RegisterUserDevice(ctx context.Context, userID uuid.UUID, fingerprint, publicKey, deviceName, platform string, isVerified bool, lastIp string) error {
	device := model.UserDevice{
		UserID: userID, DeviceFingerprint: fingerprint, PublicKey: publicKey,
		DeviceNameEncrypted: deviceName, Platform: platform, IsVerified: isVerified,
		LastIp: lastIp, IsActive: true,
	}
	return s.userRepo.SaveDevice(ctx, &device)
}

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
