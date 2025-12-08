package handler

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/http-server/config"
	"github.com/zerodayz7/http-server/internal/errors"
	"github.com/zerodayz7/http-server/internal/features/auth/service"
	"github.com/zerodayz7/http-server/internal/shared/logger"
	"github.com/zerodayz7/http-server/internal/shared/security"
	"github.com/zerodayz7/http-server/internal/validator"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// LOGIN
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	log := logger.GetLogger()
	body := c.Locals("validatedBody").(validator.LoginRequest)

	log.InfoObj("Login attempt", map[string]any{"email": body.Email})

	user, err := h.authService.GetUserByEmail(body.Email)
	if err != nil {
		log.WarnObj("User not found", map[string]any{"email": body.Email})
		return errors.SendAppError(c, errors.ErrInvalidCredentials)
	}

	valid, err := h.authService.VerifyPassword(user, body.Password)
	if err != nil || !valid {
		log.WarnObj("Invalid password", map[string]any{"userID": user.ID})
		return errors.SendAppError(c, errors.ErrInvalidCredentials)
	}

	// 2FA
	if user.TwoFactorEnabled && user.TwoFactorSecret != "" {
		return c.JSON(fiber.Map{"2fa_required": true})
	}

	accessToken, err := security.GenerateAccessToken(fmt.Sprint(user.ID))
	if err != nil {
		log.ErrorObj("Failed to generate access token", err)
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
		"expires_at":    refreshToken.ExpiresAt,
	}

	// logowanie odpowiedzi
	log.InfoMap("Login response", response)

	return c.JSON(response)
}

// REFRESH TOKEN
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	body := c.Locals("validatedBody").(validator.RefreshTokenRequest)
	accessTTL := config.AppConfig.JWT.AccessTTL
	log := logger.GetLogger()

	rt, err := h.authService.GetRefreshToken(body.RefreshToken)
	if err != nil || rt.Revoked || rt.ExpiresAt.Before(time.Now()) {
		log.WarnObj("Invalid refresh token", map[string]any{"token": body.RefreshToken})
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	accessToken, err := security.GenerateAccessToken(fmt.Sprint(rt.UserID))
	if err != nil {
		log.ErrorObj("Failed to generate access token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	return c.JSON(fiber.Map{
		"access_token": accessToken,
		"expires_at":   time.Now().Add(accessTTL),
	})
}

// LOGOUT
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	log := logger.GetLogger()
	body := c.Locals("validatedBody").(validator.RefreshTokenRequest)

	log.InfoObj("Logout attempt", map[string]any{"refresh_token": body.RefreshToken})

	err := h.authService.RevokeRefreshToken(body.RefreshToken)
	if err != nil {
		log.ErrorObj("Failed to revoke refresh token", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	log.InfoObj("Logout successful", map[string]any{"refresh_token": body.RefreshToken})

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
