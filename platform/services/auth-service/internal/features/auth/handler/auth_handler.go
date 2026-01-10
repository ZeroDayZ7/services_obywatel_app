package handler

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/events"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/types"
	"github.com/zerodayz7/platform/pkg/utils"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/http"
	service "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/service"
	"github.com/zerodayz7/platform/services/auth-service/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	authService *service.AuthService
	cache       *redis.Cache
	cfg         *types.Config
}

func NewAuthHandler(authService *service.AuthService, cache *redis.Cache, cfg *types.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		cache:       cache,
		cfg:         cfg,
	}
}

// #region REGISTER DEVICE
func (h *AuthHandler) RegisterDevice(c *fiber.Ctx) error {
	log := shared.GetLogger()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	body, ok := c.Locals("validatedBody").(validator.RegisterDeviceRequest)
	if !ok {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	sessionID := c.Get("X-Session-Id")

	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	// 2. Sprawdzamy obecny stan sesji
	var oldFingerprint string
	if sessionID != "" {
		currentSession, err := h.cache.GetSession(ctx, sessionID)
		if err == nil && currentSession != nil {
			oldFingerprint = currentSession.Fingerprint
		}
	}

	clientIP := c.Get("X-Real-IP")
	if clientIP == "" {
		clientIP = c.IP()
	}
	// ==========================================================
	// 🔐 NOWA SEKCJA: KRYPTOGRAFICZNA WERYFIKACJA URZĄDZENIA
	// ==========================================================

	// 1. Pobierz challenge z Redis (poprawiony błąd "multiple-value")
	challengeKey := fmt.Sprintf("challenge:%d", userID)
	storedChallenge, err := h.cache.Get(ctx, challengeKey)
	if err != nil {
		log.Warn(fmt.Sprintf("Challenge expired or missing for user: %d", userID))
		return errors.SendAppError(c, errors.ErrSessionExpired)
	}

	// 2. Dekoduj klucz publiczny i podpis
	pubKeyBytes, err := base64.StdEncoding.DecodeString(body.PublicKey)
	if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
		return errors.SendAppError(c, errors.ErrInvalidPairingData)
	}

	// Używamy body.Signature
	sigBytes, err := base64.StdEncoding.DecodeString(body.Signature)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidPairingData)
	}

	// 3. Weryfikacja Ed25519
	if !ed25519.Verify(pubKeyBytes, []byte(storedChallenge), sigBytes) {
		log.Error(fmt.Sprintf("Signature mismatch for user: %d", userID))
		return errors.SendAppError(c, errors.ErrVerificationFailed)
	}

	// 4. Usuwamy challenge po użyciu
	h.cache.Del(ctx, challengeKey)
	// ==========================================================

	existingDevice, err := h.authService.GetDeviceByFingerprint(ctx, userID, body.DeviceFingerprint)

	if err == nil && existingDevice != nil {
		// Urządzenie znalezione (prawdopodobnie zarejestrowane wcześniej przy resecie z IsActive: false)
		// Aktywujemy je teraz, bo użytkownik przeszedł weryfikację PIN/Challenge
		err = h.authService.ActivateDevice(ctx, existingDevice.ID, body.PublicKey, body.DeviceNameEncrypted)
		if err != nil {
			log.ErrorObj("Failed to activate existing device", err)
			return errors.SendAppError(c, errors.ErrInternal)
		}
	} else {
		// Całkowicie nowe urządzenie -> pełna rejestracja
		err = h.authService.RegisterUserDevice(
			ctx,
			userID,
			body.DeviceFingerprint,
			body.PublicKey,
			body.DeviceNameEncrypted,
			body.Platform,
			true,
			clientIP,
		)
		if err != nil {
			log.ErrorObj("RegisterDevice: Service error", err)
			return errors.SendAppError(c, errors.ErrInternal)
		}
	}

	// 3️⃣ Jeśli to pierwsze "prawdziwe" parowanie, aktualizujemy stare Refresh Tokeny
	if oldFingerprint != "" && oldFingerprint != body.DeviceFingerprint {
		_ = h.authService.UpdateRefreshTokensFingerprint(uuid.UUID(userID), oldFingerprint, body.DeviceFingerprint)
	}

	// 4️⃣ Synchronizacja Redis
	if sessionID != "" {
		if err := h.cache.UpdateSessionFingerprint(ctx, sessionID, body.DeviceFingerprint); err != nil {
			log.ErrorObj("Failed to sync Redis", err)
			return errors.SendAppError(c, errors.ErrInternal)
		}
	}

	// 5️⃣ Generujemy NOWY Access Token z NOWYM fingerprintem
	newAccessToken, _, err := h.authService.CreateAccessToken(uuid.UUID(userID), body.DeviceFingerprint)
	if err != nil {
		log.ErrorObj("Failed to issue post-registration token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 1. Pobieramy pełne dane użytkownika (żeby mieć imię, role itp.)
	user, err := h.authService.GetUserByID(
		c.Context(),
		userID,
	)
	if err != nil {
		log.ErrorObj("User not found during registration", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 2. Generujemy Refresh Token (bo po 2FA go nie było!)
	refreshToken, err := h.authService.CreateRefreshToken(userID, body.DeviceFingerprint)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	log.DebugMap("Device registration successful", map[string]any{
		"userId": userID,
		"fpt":    body.DeviceFingerprint,
	})

	userData := http.DeviceUserData{
		UserID:      user.ID.String(),
		Email:       user.Email,
		DisplayName: user.Username,
		Role:        "USER",
		LastLogin:   time.Now().Format(time.RFC3339),
		Roles:       []string{"USER"},
	}

	rbacData := map[string]any{
		"roles":       []string{"USER", "ADMIN"},
		"permissions": []string{},
	}

	response := http.RegisterDeviceResponse{
		Success:      true,
		Message:      "Device registered",
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken.Token,
		IsTrusted:    true,
		User:         userData,
		Rbac:         rbacData,
	}

	log.DebugResponse("Device registration success", response)

	emitter := events.NewEmitter(h.cache, "auth-service")

	emitter.Emit(
		ctx,
		events.DeviceRegistered,
		userID.String(),
		events.WithIP(clientIP),
		events.WithMetadata(map[string]any{
			"device":   body.DeviceNameEncrypted,
			"platform": body.Platform,
		}),
		events.WithFlags(events.EventFlags{
			Audit:  true,
			Notify: true,
		}),
	)

	return c.JSON(response)
}

// #region LOGIN
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()
	log := shared.GetLogger()

	// 1. Pobieramy body (używając Twojego validatora)
	body := c.Locals("validatedBody").(validator.LoginRequest)
	fingerprint := c.Get("X-Device-Fingerprint")

	// Czyszczenie hasła z RAM po zakończeniu funkcji
	defer func() {
		if len(body.Password) > 0 {
			for i := range body.Password {
				body.Password[i] = 0
			}
			log.Debug("Sensitive password bytes cleared from RAM")
		}
	}()

	// 2. Pobieramy użytkownika (serwis zwraca model z ID typu uuid.UUID)
	user, err := h.authService.GetUserByEmail(ctx, body.Email)
	if err != nil {
		log.WarnObj("User not found", body.Email)
		return errors.SendAppError(c, errors.ErrInvalidCredentials)
	}

	if err := h.authService.CanUserLogin(user); err != nil {
		return errors.SendAppError(c, err)
	}

	const maxFailedAttempts = 5
	if user.FailedLoginAttempts >= maxFailedAttempts {
		log.WarnObj("User exceeded failed login attempts", user.Email)
		return errors.SendAppError(c, errors.ErrAccountLocked)
	}

	valid, err := h.authService.VerifyPassword(user, body.Password)
	if err != nil || !valid {
		log.WarnObj("Invalid password", user.Email)

		_ = h.authService.IncrementUserFailedLogin(user.ID)
		return errors.SendAppError(c, errors.ErrInvalidCredentials)
	}

	// 3. Obsługa 2FA
	if user.TwoFactorEnabled {
		code := fmt.Sprintf("%06d", shared.RandInt(100000, 999999))
		hashedCode, _ := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		twoFAToken := shared.GenerateUuidV7()

		session := redis.TwoFASession{
			UserID:      fmt.Sprint(user.ID),
			Email:       user.Email,
			CodeHash:    string(hashedCode),
			Token:       twoFAToken,
			Fingerprint: fingerprint,
			Attempts:    0,
		}

		data, _ := json.Marshal(session)
		key := fmt.Sprintf("login:2fa:%s", twoFAToken)

		if err := h.cache.Set(c.Context(), key, data, 5*time.Minute); err != nil {
			log.ErrorObj("Failed to save 2FA session", err)
			return errors.SendAppError(c, errors.ErrInternal)
		}

		response := http.LoginResponse{
			TwoFARequired: true,
			TwoFAToken:    twoFAToken,
		}

		log.DebugMap("Generated 2FA code", map[string]any{
			"email": body.Email,
			"test":  code,
			"token": twoFAToken,
		})

		return c.JSON(response)
	}

	// 4. Generowanie Tokenów (Używamy user.ID bezpośrednio jako uuid.UUID)
	// To rozwiązuje błędy IncompatibleAssign
	accessToken, sessionID, err := h.authService.CreateAccessToken(user.ID, fingerprint)
	if err != nil {
		log.ErrorObj("Failed to generate access token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 5. Zapis sesji w Redis (Używamy .String() dla identyfikatora użytkownika)
	if err := h.cache.SetSession(c.Context(), sessionID, fmt.Sprint(user.ID), fingerprint, h.cfg.Session.TTL); err != nil {
		log.ErrorObj("Failed to save session in Redis", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	refreshToken, err := h.authService.CreateRefreshToken(user.ID, fingerprint)
	if err != nil {
		log.ErrorObj("Failed to create refresh token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 6. Przygotowanie odpowiedzi dla Fluttera
	response := http.LoginResponse{
		TwoFARequired: false,
		AccessToken:   accessToken,
		RefreshToken:  refreshToken.Token,
		UserID:        fmt.Sprint(user.ID),
		ExpiresAt:     refreshToken.ExpiresAt.Unix(),
	}

	log.InfoMap("Login response successful", map[string]any{"user": user.Email})

	// Wysyłka gotowej struktury
	return c.JSON(response)
}

// #region VERIFY 2 FA
func (h *AuthHandler) Verify2FA(c *fiber.Ctx) error {
	// ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	// defer cancel()
	log := shared.GetLogger()

	// 1. Pobieramy body z Locals (walidowane przez middleware)
	body, ok := c.Locals("validatedBody").(validator.TwoFARequest)
	if !ok {
		log.Error("Failed to cast validatedBody to TwoFARequest")
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 2. POBIERAMY FINGERPRINT Z NAGŁÓWKA
	// To rozwiązuje błąd "not enough arguments in call to h.authService.CreateAccessToken"
	fingerprint := c.Get("X-Device-Fingerprint")

	// ✅ KLUCZOWE: Zerowanie kodu 2FA z pamięci na koniec funkcji
	defer func() {
		if len(body.Code) > 0 {
			for i := range body.Code {
				body.Code[i] = 0
			}
			log.Debug("Sensitive 2FA code bytes cleared from RAM")
		}
	}()

	// 3. Pobieranie sesji 2FA z Cache
	key := fmt.Sprintf("login:2fa:%s", body.Token)
	data, err := h.cache.Get(c.Context(), key)
	if err != nil {
		log.WarnObj("2FA session expired or not found", map[string]string{"token": body.Token})
		return errors.SendAppError(c, errors.ErrInvalidCredentials)
	}

	var session redis.TwoFASession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		log.ErrorObj("Failed to unmarshal 2FA session", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 5. Weryfikacja kodu bcryptem
	if err := bcrypt.CompareHashAndPassword([]byte(session.CodeHash), body.Code); err != nil {

		status, err := h.cache.Verify2FAAttempt(
			c.Context(),
			key,
			5,
			5*time.Minute,
		)
		if err != nil {
			log.ErrorObj("Verify2FAAttempt failed", err)
			return errors.SendAppError(c, errors.ErrInternal)
		}

		switch status {
		case "locked":
			return errors.SendAppError(c, errors.Err2FALocked)
		case "not_found":
			return errors.SendAppError(c, errors.ErrInvalidCredentials)
		case "attempt_updated":
			return errors.SendAppError(c, errors.ErrInvalid2FACode)
		default:
			log.ErrorMap("Unknown 2FA status", map[string]any{"status": status})
			return errors.SendAppError(c, errors.ErrInternal)
		}
	}

	// 6. Usuwamy sesję 2FA po sukcesie
	if err := h.cache.Del(c.Context(), key); err != nil {
		log.WarnObj("Failed to cleanup 2FA session", err)
	}

	uid, err := uuid.Parse(session.UserID)
	if err != nil {
		log.ErrorMap("Błędny format UUID w sesji", map[string]any{
			"userID": session.UserID,
			"error":  err.Error(),
		})
		return errors.SendAppError(c, errors.ErrInternal)
	}

	user, err := h.authService.GetUserByID(
		c.Context(),
		uid,
	)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// Aktualizujemy LastLogin i LastIP
	user.LastLogin = time.Now()
	user.LastIP = c.IP()

	// Zapisujemy zmiany w bazie poprzez repo w AuthService
	if err := h.authService.RepoUpdateUser(
		c.Context(),
		user,
	); err != nil {
		log.ErrorObj("Failed to update LastLogin/LastIP", err)
	}

	// 7. TWORZYMY TOKENY (Przekazujemy uid ORAZ fingerprint)
	accessToken, sessionID, err := h.authService.CreateAccessToken(uid, fingerprint)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 8. ZAPISUJEMY SESJĘ W REDIS
	if err := h.cache.SetSession(c.Context(), sessionID, session.UserID, fingerprint, h.cfg.Session.TTL); err != nil {
		log.ErrorObj("Failed to set session in Redis", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	challenge := shared.GenerateUuid()
	h.cache.Set(c.Context(), fmt.Sprintf("challenge:%d", uid), challenge, 5*time.Minute)

	response := http.Verify2FAResponse{
		Success:     true,
		AccessToken: accessToken,
		Challenge:   challenge,
		IsTrusted:   false,
	}

	log.InfoMap("2FA verification successful", map[string]any{
		"user_id": uid,
		"token":   sessionID,
	})
	return c.JSON(response)
}

// #region REFRESH TOKEN
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	// ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	// defer cancel()
	body := c.Locals("validatedBody").(validator.RefreshTokenRequest)
	log := shared.GetLogger()

	// 1. Pobieramy fingerprint z nagłówka
	// Musi być wysłany przez klienta (np. Interceptor Dio we Flutterze)
	fingerprint := c.Get("X-Device-Fingerprint")
	if fingerprint == "" {
		log.Warn("Refresh attempt without fingerprint header")
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	// 2. Pobranie Refresh Tokena z bazy danych
	rt, err := h.authService.GetRefreshToken(body.RefreshToken)
	if err != nil || rt.Revoked || rt.ExpiresAt.Before(time.Now()) {
		log.WarnObj("Invalid, revoked or expired refresh token", body.RefreshToken)
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	// --- KLUCZOWA ZMIANA: Weryfikacja powiązania z urządzeniem ---
	// Sprawdzamy, czy ten token należy do urządzenia, które o niego prosi
	if rt.DeviceFingerprint != fingerprint {
		log.WarnMap("SECURITY ALERT: Refresh token used on different device!", map[string]any{
			"user_id":      rt.UserID,
			"expected_fpt": rt.DeviceFingerprint,
			"received_fpt": fingerprint,
		})
		// Opcjonalnie: h.authService.RevokeAllUserTokens(rt.UserID) - dla maksymalnego bezpieczeństwa
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	userIDStr := fmt.Sprint(rt.UserID)

	// 3. Generujemy nowy Access Token (z nowym sessionID i fingerprintem w środku JWT)
	accessToken, sessionID, err := h.authService.CreateAccessToken(rt.UserID, fingerprint)
	if err != nil {
		log.ErrorObj("Failed to generate access token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 4. Tworzymy NOWĄ sesję w Redis
	// To SID wlatuje do JWT, a Gateway sprawdzi to przy następnym żądaniu
	if err := h.cache.SetSession(c.Context(), sessionID, userIDStr, fingerprint, h.cfg.Session.TTL); err != nil {
		log.ErrorObj("Failed to save session in Redis", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 5. Odpowiedź JSON
	response := http.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: rt.Token,
		UserID:       userIDStr,
		ExpiresAt:    time.Now().Add(h.cfg.JWT.AccessTTL).Unix(),
	}

	log.InfoMap("Token refreshed successfully", map[string]any{
		"user_id":    userIDStr,
		"session_id": sessionID,
		"fpt":        fingerprint,
	})

	return c.JSON(response)
}

// #region LOGOUT
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	log := shared.GetLogger()

	// 1. Pobierz body (refresh token)
	body := c.Locals("validatedBody").(validator.RefreshTokenRequest)
	if body.RefreshToken == "" {
		log.Warn("Missing refresh token in request body")
		return fiber.NewError(fiber.StatusBadRequest, "Missing refresh token")
	}

	// 2. Pobierz X-User-Id i X-Session-Id z headerów
	userID := c.Get("X-User-Id")
	sessionID := c.Get("X-Session-Id")
	if userID == "" || sessionID == "" {
		log.Warn("Missing X-User-Id or X-Session-Id headers")
		return fiber.NewError(fiber.StatusBadRequest, "Missing required headers")
	}

	// 3. Usuń refresh token z DB
	if err := h.authService.RevokeRefreshToken(body.RefreshToken); err != nil {
		log.ErrorObj("Failed to revoke refresh token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}
	log.InfoObj("Refresh token revoked", userID)

	// 4. Pobierz userID z Redis po sessionID
	storedUserID, err := h.cache.GetUserIDBySession(c.Context(), sessionID)
	if err != nil {
		log.ErrorObj("Failed to get session from Redis", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 5. Sprawdź zgodność userID z Redis
	if storedUserID != userID {
		log.WarnObj("Session user mismatch", []string{sessionID, userID, storedUserID})
		return fiber.NewError(fiber.StatusForbidden, "Invalid session")
	}

	// 6. Usuń sesję z Redis
	if err := h.cache.DeleteSession(c.Context(), sessionID); err != nil {
		log.ErrorObj("Failed to delete session from Redis", err)
	} else {
		log.InfoObj("Session deleted from Redis", sessionID)
	}

	response := http.LogoutResponse{
		Message: "Logged out successfully",
	}

	log.InfoMap("Logout successful", map[string]any{
		"user_id":    userID,
		"session_id": sessionID,
	})

	return c.JSON(response)
}

// #region REGISTER
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(validator.RegisterRequest)

	// Próba rejestracji użytkownika
	user, err := h.authService.Register(body.Username, body.Email, body.Password)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			errors.AttachRequestMeta(c, appErr, "requestID")
			return appErr
		}
		return errors.ErrInternal
	}

	// Przygotowanie minimalistycznej odpowiedzi DTO
	response := http.RegisterResponse{
		Success: true,
	}

	// Logujemy fakt rejestracji (zachowując ID w logach serwera dla audytu)
	log.InfoMap("User account created successfully", map[string]any{
		"email":   user.Email,
		"user_id": user.ID,
	})

	return c.Status(fiber.StatusCreated).JSON(response)
}
