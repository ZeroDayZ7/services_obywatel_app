package handler

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/service"
	"github.com/zerodayz7/platform/services/auth-service/internal/validator"
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
	if user.TwoFactorEnabled && user.TwoFactorSecret != "" {
		return c.JSON(fiber.Map{"2fa_required": true})
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
		"refresh_token": rt.Token, // bierzemy token z bazy
		"2fa_required":  false,
	}

	return c.JSON(response)
}

// LOGOUT
// handler/auth_handler.go
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	log := shared.GetLogger()

	body := c.Locals("validatedBody").(validator.RefreshTokenRequest)

	// 1. Odczyt SID z JWT
	jwtToken := c.Locals("user").(*jwt.Token)
	claims := jwtToken.Claims.(jwt.MapClaims)

	sessionID, _ := claims["sid"].(string)

	// 2. Usuń refresh token z DB
	if err := h.authService.RevokeRefreshToken(body.RefreshToken); err != nil {
		log.ErrorObj("Failed to revoke refresh token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 3. Usuń sesję z Redis
	if sessionID != "" {
		if err := h.cache.DeleteSession(c.Context(), sessionID); err != nil {
			log.ErrorObj("Failed to delete session from Redis", err)
		} else {
			log.InfoObj("Session deleted from Redis", map[string]any{"session_id": sessionID})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
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

// OPTIONAL: Verify2FA jeśli używasz 2FA
func (h *AuthHandler) Verify2FA(c *fiber.Ctx) error {
	body := c.Locals("validatedBody").(validator.TwoFARequest)
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return errors.SendAppError(c, errors.ErrUnauthorized)
	}
	ok, err := h.authService.Verify2FACodeByID(userID, body.Code)
	if err != nil || !ok {
		return errors.SendAppError(c, errors.ErrInvalid2FACode)
	}
	return c.JSON(fiber.Map{"message": "2FA verified successfully"})
}
