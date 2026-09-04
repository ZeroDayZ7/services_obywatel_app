package router

import (
	"github.com/gofiber/fiber/v2"

	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/pkg/shared"

	handler "github.com/zerodayz7/platform/services/auth-service/internal/handler"
	"github.com/zerodayz7/platform/services/auth-service/internal/middleware"
)

//#region SetupAuthRoutes
func SetupAuthRoutes(
	app *fiber.App,
	h *handler.AuthHandler,
	resetHandler *handler.ResetHandler,
) {
	auth := app.Group("/auth")
	auth.Use(shared.GetLimiter(shared.LimitAuth, nil))
	// ==========================
	// LOGIN / REGISTER / JWT
	// ==========================
	auth.Post("/login",
		middleware.ValidateBody[schemas.LoginRequest](),
		h.Login,
	)
	

	auth.Post("/login/step2",
		middleware.ValidateBody[schemas.LoginStep2Request](),
		h.LoginStep2,
	)
	

	auth.Post("/2fa-verify",
		middleware.ValidateBody[schemas.TwoFARequest](),
		h.Verify2FA,
	)

	auth.Post("/2fa-resend",
		middleware.ValidateBody[schemas.ResendTwoFARequest](),
		h.Resend2FA,
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
	// DEVICE MANAGEMENT
	// ==========================
	auth.Post("/register-device",
		middleware.ValidateBody[schemas.RegisterDeviceRequest](),
		h.RegisterDevice,
	)

	auth.Post("/temporary-session", h.CreateTemporarySession)

	auth.Post("/verify-device",
		middleware.ValidateBody[schemas.VerifyDeviceRequest](),
		h.VerifyDevice,
	)

	auth.Post("/unpair-device", h.UnpairDevice)

	// ==========================
	// RESET PASSWORD
	// ==========================
	reset := auth.Group("/reset")
	reset.Use(shared.GetLimiter(shared.LimitReset, nil))

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
