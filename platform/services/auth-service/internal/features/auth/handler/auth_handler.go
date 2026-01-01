package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/service"
	"github.com/zerodayz7/platform/services/auth-service/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	authService *service.AuthService
	cache       *redis.Cache
}

func NewAuthHandler(authService *service.AuthService, cache *redis.Cache) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		cache:       cache,
	}
}

type TwoFASession struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	CodeHash    string `json:"code_hash"`
	Token       string `json:"token"`
	Fingerprint string `json:"fingerprint"`
	Attempts    int    `json:"attempts"`
}

// LOGIN
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	log := shared.GetLogger()

	body := c.Locals("validatedBody").(validator.LoginRequest)

	// 1. POBIERAMY FINGERPRINT Z NAGŁÓWKA
	fingerprint := c.Get("X-Device-Fingerprint")

	defer func() {
		if len(body.Password) > 0 {
			for i := range body.Password {
				body.Password[i] = 0
			}
			log.Debug("Sensitive password bytes cleared from RAM")
		}
	}()

	user, err := h.authService.GetUserByEmail(body.Email)
	if err != nil {
		log.WarnObj("User not found", body)
		return errors.SendAppError(c, errors.ErrInvalidCredentials)
	}

	valid, err := h.authService.VerifyPassword(user, body.Password)
	if err != nil || !valid {
		log.WarnObj("Invalid password", user)
		return errors.SendAppError(c, errors.ErrInvalidCredentials)
	}

	// 2FA - Jeśli włączone, zwracamy token sesji 2FA i kończymy
	if user.TwoFactorEnabled {
		code := fmt.Sprintf("%06d", shared.RandInt(100000, 999999))
		hashedCode, _ := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		twoFAToken := shared.GenerateUuid()

		log.DebugMap("Generated 2FA code", map[string]any{
			"email": body.Email,
			"code":  code,
			"token": twoFAToken,
		})

		session := TwoFASession{
			UserID:      fmt.Sprint(user.ID),
			Email:       user.Email,
			CodeHash:    string(hashedCode),
			Token:       twoFAToken,
			Fingerprint: fingerprint,
			Attempts:    0,
		}

		data, err := json.Marshal(session)
		if err != nil {
			log.ErrorObj("Failed to marshal 2FA session", err)
			return errors.SendAppError(c, errors.ErrInternal)
		}

		key := fmt.Sprintf("login:2fa:%s", twoFAToken)
		ttl := 5 * time.Minute

		if err := h.cache.Set(c.Context(), key, data, ttl); err != nil {
			log.ErrorObj("Failed to save 2FA session", err)
			return errors.SendAppError(c, errors.ErrInternal)
		}

		return c.JSON(fiber.Map{
			"2fa_required": true,
			"two_fa_token": twoFAToken,
		})
	}

	// --- LOGOWANIE BEZPOŚREDNIE (Jeśli 2FA wyłączone) ---
	userIDStr := fmt.Sprint(user.ID)

	// 2. TWORZYMY TOKEN (Z FINGERPRINTEM)
	// Naprawia błąd kompilacji: want (uint, string)
	accessToken, sessionID, err := h.authService.CreateAccessToken(user.ID, fingerprint)
	if err != nil {
		log.ErrorObj("Failed to generate access token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 3. ZAPISUJEMY SESJĘ W REDIS
	if err := h.cache.SetSession(c.Context(), sessionID, userIDStr, fingerprint, config.AppConfig.SessionTTL); err != nil {
		log.ErrorObj("Failed to save session in Redis", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	refreshToken, err := h.authService.CreateRefreshToken(user.ID, fingerprint)
	if err != nil {
		log.ErrorObj("Failed to create refresh token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	response := fiber.Map{
		"2fa_required":  false,
		"access_token":  accessToken,
		"refresh_token": refreshToken.Token,
		"user_id":       userIDStr,
		"expires_at":    refreshToken.ExpiresAt,
	}

	log.InfoMap("Login response successful with device binding", response)
	return c.JSON(response)
}

func (h *AuthHandler) Verify2FA(c *fiber.Ctx) error {
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

	var session TwoFASession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		log.ErrorObj("Failed to unmarshal 2FA session", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 4. Sprawdzamy limit prób
	if session.Attempts >= 5 {
		log.WarnObj("Max 2FA attempts reached", map[string]string{"user_id": session.UserID})
		return errors.SendAppError(c, errors.ErrInvalid2FACode)
	}

	// 5. Weryfikacja kodu bcryptem
	if err := bcrypt.CompareHashAndPassword([]byte(session.CodeHash), body.Code); err != nil {
		session.Attempts++
		updated, _ := json.Marshal(session)
		h.cache.Set(c.Context(), key, updated, 5*time.Minute)
		return errors.SendAppError(c, errors.ErrInvalid2FACode)
	}

	// 6. Usuwamy sesję 2FA po sukcesie
	h.cache.Del(c.Context(), key)

	userID, err := strconv.ParseUint(session.UserID, 10, 64)
	if err != nil {
		log.ErrorObj("Failed to parse user ID from session", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}
	uid := uint(userID)

	// 7. TWORZYMY TOKENY (Przekazujemy uid ORAZ fingerprint)
	accessToken, sessionID, err := h.authService.CreateAccessToken(uid, fingerprint)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 8. ZAPISUJEMY SESJĘ W REDIS
	if err := h.cache.SetSession(c.Context(), sessionID, session.UserID, fingerprint, config.AppConfig.SessionTTL); err != nil {
		log.ErrorObj("Failed to set session in Redis", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	refreshToken, err := h.authService.CreateRefreshToken(uid, fingerprint)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	response := fiber.Map{
		"success":       true,
		"access_token":  accessToken,
		"refresh_token": refreshToken.Token,
		"user_id":       session.UserID,
		"expires_at":    refreshToken.ExpiresAt,
	}

	log.InfoMap("2FA verification successful", response)
	return c.JSON(response)
}

// REFRESH TOKEN
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
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
	if err := h.cache.SetSession(c.Context(), sessionID, userIDStr, fingerprint, config.AppConfig.SessionTTL); err != nil {
		log.ErrorObj("Failed to save session in Redis", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 5. Odpowiedź JSON
	response := fiber.Map{
		"access_token":  accessToken,
		"refresh_token": rt.Token,
		"user_id":       userIDStr,
		"expires_at":    time.Now().Add(config.AppConfig.JWT.AccessTTL).Unix(),
	}

	log.InfoMap("Token refreshed successfully", map[string]any{
		"user_id":    userIDStr,
		"session_id": sessionID,
		"fpt":        fingerprint,
	})

	return c.JSON(response)
}

// LOGOUT
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

	// Tworzymy odpowiedź
	response := fiber.Map{
		"message": "Logged out successfully",
	}

	// Dodawanie kolejnych pól w razie potrzeby
	// response["user_id"] = userID
	// response["events"] = []string{"refresh_token_revoked", "session_deleted"}

	// Zwracamy odpowiedź
	return c.JSON(response)
}

// REGISTER
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	body := c.Locals("validatedBody").(validator.RegisterRequest)

	user, err := h.authService.Register(body.Username, body.Email, body.Password)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			errors.AttachRequestMeta(c, appErr, "requestID")
			return appErr
		}
		return errors.ErrInternal
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"user":    user,
	})
}

// RegisterDevice obsługuje zapisanie klucza publicznego urządzenia i jego danych
func (h *AuthHandler) RegisterDevice(c *fiber.Ctx) error {
	log := shared.GetLogger()

	body, ok := c.Locals("validatedBody").(validator.RegisterDeviceRequest)
	if !ok {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 1. Wyciągamy dane sesyjne
	userIDStr := c.Get("X-User-Id")
	sessionID := c.Get("X-Session-Id") // Gateway musi to przekazywać!

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	// 2. Zapisujemy NOWY fingerprint do bazy SQL (ten z body)
	err = h.authService.RegisterUserDevice(
		c.Context(),
		uint(userID),
		body.DeviceFingerprint, // To jest ten docelowy: bae39e...
		body.PublicKey,
		body.DeviceNameEncrypted,
		body.Platform,
	)

	if err != nil {
		log.ErrorObj("RegisterDevice: Service error", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 3. 🔥 KLUCZOWA SYNCHRONIZACJA REDIS 🔥
	// Od tego momentu Flutter (przez Interceptor) będzie wysyłał NOWY fingerprint.
	// Musimy go zaktualizować w sesji, żeby Middleware Gatewaya nas nie wyrzuciło.
	if sessionID != "" {
		err = h.cache.UpdateSessionFingerprint(c.Context(), sessionID, body.DeviceFingerprint)
		if err != nil {
			log.ErrorObj("Failed to sync new fingerprint to Redis", err)
			// Możesz tu zwrócić błąd, bo bez tego kolejne żądanie padnie na 401
			return errors.SendAppError(c, errors.ErrInternal)
		}
		log.Info("Session fingerprint updated in Redis to new version")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Device registered and session synced",
	})
}
