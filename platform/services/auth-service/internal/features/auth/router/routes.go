package router

import (
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/handler"
	"github.com/zerodayz7/platform/services/auth-service/internal/middleware"
	"github.com/zerodayz7/platform/services/auth-service/internal/validator"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/auth-service/config"
)

func SetupAuthRoutes(app *fiber.App, h *handler.AuthHandler) {
	auth := app.Group("/auth")
	auth.Use(config.NewLimiter("auth"))

	auth.Post("/login",
		middleware.ValidateBody[validator.LoginRequest](),
		h.Login,
	)

	auth.Post("/2fa-verify",
		middleware.ValidateBody[validator.TwoFARequest](),
		h.Verify2FA)

	auth.Post("/register",
		middleware.ValidateBody[validator.RegisterRequest](),
		h.Register,
	)
	// ==========================
	// JWT-specific routes
	// ==========================

	// Endpoint do odświeżania Access Token
	auth.Post("/refresh",
		middleware.ValidateBody[validator.RefreshTokenRequest](),
		h.RefreshToken,
	)

	// Opcjonalnie endpoint do wylogowania (unieważnienia Refresh Token)
	auth.Post("/logout",
		middleware.ValidateBody[validator.RefreshTokenRequest](),
		h.Logout,
	)
}
