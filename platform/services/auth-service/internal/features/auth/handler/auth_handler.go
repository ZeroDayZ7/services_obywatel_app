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
	UserID   string `json:"user_id"`  // ID użytkownika
	Email    string `json:"email"`    // opcjonalnie email do wysyłki powiadomień
	CodeHash string `json:"code"`     // kod 2FA
	Token    string `json:"token"`    // token wygenerowany do 2FA
	Attempts int    `json:"attempts"` // liczba prób
}

// LOGIN
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(validator.LoginRequest)

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

	// 2FA
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
			UserID:   fmt.Sprint(user.ID),
			Email:    user.Email,
			CodeHash: string(hashedCode),
			Token:    twoFAToken,
			Attempts: 0,
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

		// wysyłka maila/SMS
		// h.authService.Send2FACode(user.Email, code)

		return c.JSON(fiber.Map{
			"2fa_required": true,
			"two_fa_token": twoFAToken,
		})
	}

	userID := fmt.Sprint(user.ID)

	// generujemy access token i sessionID
	accessToken, sessionID, err := h.authService.CreateAccessToken(user.ID)
	if err != nil {
		log.ErrorObj("Failed to generate access token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// zapis w Redis
	if err := h.cache.SetSession(c.Context(), sessionID, userID, config.AppConfig.SessionTTL); err != nil {
		log.ErrorObj("Failed to save session in Redis", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	refreshToken, err := h.authService.CreateRefreshToken(user.ID)
	if err != nil {
		log.ErrorObj("Failed to create refresh token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	response := fiber.Map{
		"2fa_required":  false,
		"access_token":  accessToken,
		"refresh_token": refreshToken.Token,
		"user_id":       userID,
		"expires_at":    refreshToken.ExpiresAt,
	}

	log.InfoMap("Login response", response)
	return c.JSON(response)
}

// REFRESH TOKEN
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	body := c.Locals("validatedBody").(validator.RefreshTokenRequest)
	log := shared.GetLogger()

	// pobranie refresh tokena z bazy
	rt, err := h.authService.GetRefreshToken(body.RefreshToken)
	if err != nil || rt.Revoked || rt.ExpiresAt.Before(time.Now()) {
		log.WarnObj("Invalid refresh token", body)
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	userID := fmt.Sprint(rt.UserID)

	// generujemy nowy access token + sessionID
	accessToken, sessionID, err := h.authService.CreateAccessToken(rt.UserID)
	if err != nil {
		log.ErrorObj("Failed to generate access token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// zapis sessionID w Redis
	if err := h.cache.SetSession(c.Context(), sessionID, userID, config.AppConfig.SessionTTL); err != nil {
		log.ErrorObj("Failed to save session in Redis", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// tworzymy odpowiedź
	response := fiber.Map{
		"access_token":  accessToken,
		"expires_at":    time.Now().Add(config.AppConfig.JWT.AccessTTL),
		"user_id":       userID,
		"refresh_token": rt.Token,
		"2fa_required":  false,
	}

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

func (h *AuthHandler) Verify2FA(c *fiber.Ctx) error {
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(validator.TwoFARequest)

	key := fmt.Sprintf("login:2fa:%s", body.Token)
	data, err := h.cache.Get(c.Context(), key)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	var session TwoFASession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	if session.Attempts >= 5 {
		return errors.SendAppError(c, errors.ErrInvalid2FACode)
	}

	if bcrypt.CompareHashAndPassword([]byte(session.CodeHash), []byte(body.Code)) != nil {
		session.Attempts++
		updated, _ := json.Marshal(session)
		h.cache.Set(c.Context(), key, updated, 5*time.Minute)
		return errors.SendAppError(c, errors.ErrInvalid2FACode)
	}

	h.cache.Del(c.Context(), key)

	userID, err := strconv.ParseUint(session.UserID, 10, 64)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}
	uid := uint(userID)

	accessToken, _, _ := h.authService.CreateAccessToken(uid)
	refreshToken, _ := h.authService.CreateRefreshToken(uid)

	response := fiber.Map{
		"success":       true,
		"access_token":  accessToken,
		"refresh_token": refreshToken.Token,
		"user_id":       session.UserID,
	}
	log.InfoMap("Login response", response)

	// Zwracamy JSON
	return c.JSON(response)

}
