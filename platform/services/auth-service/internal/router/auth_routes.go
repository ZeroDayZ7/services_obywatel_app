package router

import (
	"github.com/gofiber/fiber/v2"

	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/pkg/shared"

	handler "github.com/zerodayz7/platform/services/auth-service/internal/handler"
	"github.com/zerodayz7/platform/services/auth-service/internal/middleware"
)

func SetupAuthRoutes(
	app *fiber.App,
	h *handler.AuthHandler,
	resetHandler *handler.ResetHandler,
) {
	auth := app.Group("/auth")
	auth.Use(shared.NewLimiter("auth", nil))

	// ==========================
	// LOGIN / REGISTER / JWT
	// ==========================
	auth.Post("/login",
		middleware.ValidateBody[schemas.LoginRequest](),
		h.Login,
	)

	auth.Post("/2fa-verify",
		middleware.ValidateBody[schemas.TwoFARequest](),
		h.Verify2FA,
	)

	auth.Post("/register",
		middleware.ValidateBody[schemas.RegisterRequest](),
		h.Register,
	)

	auth.Post("/refresh",
		middleware.ValidateBody[schemas.RefreshTokenRequest](),
		h.RefreshToken,
	)

	auth.Post("/logout",
		middleware.ValidateBody[schemas.RefreshTokenRequest](),
		h.Logout,
	)

	// ==========================
	// DEVICE MANAGEMENT (NEW)
	// ==========================
	// Tutaj dodajemy endpoint, którego szuka Flutter
	auth.Post("/register-device",
		middleware.ValidateBody[schemas.RegisterDeviceRequest](),
		h.RegisterDevice,
	)

	// ==========================
	// RESET PASSWORD
	// ==========================
	reset := auth.Group("/reset")
	reset.Use(shared.NewLimiter("reset", nil))

	reset.Post("/send",
		middleware.ValidateBody[schemas.ResetPasswordRequest](),
		resetHandler.SendResetCode,
	)

	reset.Post("/verify",
		middleware.ValidateBody[schemas.ResetCodeVerifyRequest](),
		resetHandler.VerifyResetCode,
	)

	reset.Post("/final",
		middleware.ValidateBody[schemas.ResetPasswordFinalRequest](),
		resetHandler.FinalizeReset,
	)
}
