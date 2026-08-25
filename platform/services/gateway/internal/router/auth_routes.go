package router

import (
	"github.com/gofiber/fiber/v2"
	pkgMiddleware "github.com/zerodayz7/platform/pkg/middleware"
	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	"github.com/zerodayz7/platform/services/gateway/internal/handlers"
)

const ServiceAuth = "auth-service"

//#region RegisterAuthRoutes
func RegisterAuthRoutes(app *fiber.App, container *di.Container) {
	target := container.Config.Services.Auth

	// Trasy publiczne z walidacją DTO
	authPublic := app.Group("/auth")
	authPublic.Post("/login", pkgMiddleware.ValidateBody[schemas.LoginRequest](), ReverseProxy(container, ServiceAuth, target))
	authPublic.Post("/verify-device", pkgMiddleware.ValidateBody[schemas.VerifyDeviceRequest](), ReverseProxy(container, ServiceAuth, target))
	authPublic.Post("/2fa-verify", pkgMiddleware.ValidateBody[schemas.TwoFARequest](), ReverseProxy(container, ServiceAuth, target))
	authPublic.Post("/2fa-resend", pkgMiddleware.ValidateBody[schemas.ResendTwoFARequest](), ReverseProxy(container, ServiceAuth, target))
	authPublic.Post("/refresh", pkgMiddleware.ValidateBody[schemas.RefreshTokenRequest](), ReverseProxy(container, ServiceAuth, target))
	authPublic.Post("/reset/send", pkgMiddleware.ValidateBody[schemas.ResetPasswordRequest](), ReverseProxy(container, ServiceAuth, target))
	authPublic.Post("/reset/verify", pkgMiddleware.ValidateBody[schemas.ResetCodeVerifyRequest](), ReverseProxy(container, ServiceAuth, target))
	authPublic.Post("/reset/final", pkgMiddleware.ValidateBody[schemas.ResetPasswordFinalRequest](), ReverseProxy(container, ServiceAuth, target))

	// Trasy chronione
	authSecure := app.Group("/auth")
	authSecure.Get("/me", handlers.GetMeHandler)

	authSecure.Post("/unpair-device", ReverseProxySecure(container, ServiceAuth, target))
	authSecure.Post("/register-device", pkgMiddleware.ValidateBody[schemas.RegisterDeviceRequest](), ReverseProxySecure(container, ServiceAuth, target))
	authSecure.Post("/logout", pkgMiddleware.ValidateBody[schemas.RefreshTokenRequest](), ReverseProxySecure(container, ServiceAuth, target))

	// Kontekst użytkownika (przekierowany do Auth)
	userSecure := app.Group("/user")
	userSecure.Get("/sessions", ReverseProxySecure(container, ServiceAuth, target))
	userSecure.Post("/sessions/terminate", ReverseProxySecure(container, ServiceAuth, target))
}
