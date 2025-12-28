package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/handler"
	"github.com/zerodayz7/platform/services/auth-service/internal/middleware"
	"github.com/zerodayz7/platform/services/auth-service/internal/validator"
)

func SetupAuthRoutes(app *fiber.App, h *handler.AuthHandler, resetHandler *handler.ResetHandler) {
	auth := app.Group("/auth")
	auth.Use(shared.NewLimiter("auth"))


	// ==========================
	// LOGIN / REGISTER / JWT
	// ==========================
	auth.Post("/login",
		middleware.ValidateBody[validator.LoginRequest](),
		h.Login,
	)

	auth.Post("/2fa-verify",
		middleware.ValidateBody[validator.TwoFARequest](),
		h.Verify2FA,
	)

	auth.Post("/register",
		middleware.ValidateBody[validator.RegisterRequest](),
		h.Register,
	)

	auth.Post("/refresh",
		middleware.ValidateBody[validator.RefreshTokenRequest](),
		h.RefreshToken,
	)

	auth.Post("/logout",
		middleware.ValidateBody[validator.RefreshTokenRequest](),
		h.Logout,
	)

	// ==========================
	// RESET PASSWORD
	// ==========================
	reset := auth.Group("/reset")
	reset.Use(shared.NewLimiter("reset"))
	
	reset.Post("/send",
		middleware.ValidateBody[validator.ResetPasswordRequest](),
		resetHandler.SendResetCode,
	)

	reset.Post("/verify",
		middleware.ValidateBody[validator.ResetCodeVerifyRequest](),
		resetHandler.VerifyResetCode,
	)

	reset.Post("/final",
		middleware.ValidateBody[validator.ResetPasswordFinalRequest](),
		resetHandler.ResetPassword,
	)
}
