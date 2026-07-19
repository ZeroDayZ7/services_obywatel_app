package router

import (
	"github.com/gofiber/fiber/v2"
	pkgMiddleware "github.com/zerodayz7/platform/pkg/middleware"
	pkgRouter "github.com/zerodayz7/platform/pkg/router"
	"github.com/zerodayz7/platform/pkg/router/health"
	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

func SetupRoutes(app *fiber.App, container *di.Container) {
	services := container.Config.Services

	// 1. Health Checks
	checker := &health.Checker{
		Redis:   container.Redis.Client,
		Service: "gateway",
		Version: container.Config.Server.AppVersion,
		Upstreams: []string{
			services.Auth + "/health",
			services.Documents + "/health",
			services.Notify + "/health",
			services.Users + "/health",
		},
	}
	health.RegisterRoutes(app, checker)

	// --- AUTH SERVICE (Publiczne) ---
	auth := services.Auth
	app.Post("/auth/login",
		pkgMiddleware.ValidateBody[schemas.LoginRequest](),
		ReverseProxySecure(container, auth),
	)

	app.Post("/auth/verify-device",
		pkgMiddleware.ValidateBody[schemas.VerifyDeviceRequest](),
		ReverseProxySecure(container, auth),
	)

	app.Post("/auth/2fa-verify",
		pkgMiddleware.ValidateBody[schemas.TwoFARequest](),
		ReverseProxy(container, auth),
	)

	app.Post("/auth/refresh",
		pkgMiddleware.ValidateBody[schemas.RefreshTokenRequest](),
		ReverseProxy(container, auth),
	)

	app.Post("/auth/reset/send",
		pkgMiddleware.ValidateBody[schemas.ResetPasswordRequest](),
		ReverseProxy(container, auth),
	)

	app.Post("/auth/reset/verify",
		pkgMiddleware.ValidateBody[schemas.ResetCodeVerifyRequest](),
		ReverseProxy(container, auth),
	)

	app.Post("/auth/reset/final",
		pkgMiddleware.ValidateBody[schemas.ResetPasswordFinalRequest](),
		ReverseProxy(container, auth),
	)

	// --- AUTH SERVICE (Zabezpieczone) ---
	app.Post("/auth/register-device",
		pkgMiddleware.ValidateBody[schemas.RegisterDeviceRequest](),
		ReverseProxySecure(container, auth),
	)
	app.Post("/auth/logout",
		pkgMiddleware.ValidateBody[schemas.RefreshTokenRequest](),
		ReverseProxySecure(container, auth),
	)

	app.Get("/user/sessions", ReverseProxySecure(container, auth))
	app.Post("/user/sessions/terminate", ReverseProxySecure(container, auth))

	// --- NOTIFICATIONS (Zabezpieczone) ---
	notify := services.Notify
	app.All("/notifications*", ReverseProxySecure(container, notify))

	// --- DOCUMENTS (Zabezpieczone) ---
	documents := services.Documents
	app.All("/documents/*", ReverseProxySecure(container, documents))

	// --- USERS SERVICE ---
	users := services.Users
	app.All("/users/*", ReverseProxySecure(container, users))

	// Fallback (404 / 405)
	pkgRouter.SetupFallbackHandlers(app)
}
