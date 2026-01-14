package router

import (
	"github.com/gofiber/fiber/v2"
	pkgRouter "github.com/zerodayz7/platform/pkg/router"
	"github.com/zerodayz7/platform/pkg/router/health"
	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	"github.com/zerodayz7/platform/services/gateway/internal/middleware"
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
		middleware.ValidateBody[schemas.LoginRequest](),
		ReverseProxySecure(container, auth),
	)

	app.Post("/auth/verify-device",
		middleware.ValidateBody[schemas.VerifyDeviceRequest](),
		ReverseProxy(container, auth),
	)

	app.Post("/auth/2fa-verify",
		middleware.ValidateBody[schemas.TwoFARequest](),
		ReverseProxy(container, auth),
	)

	app.Post("/auth/refresh",
		middleware.ValidateBody[schemas.RefreshTokenRequest](),
		ReverseProxy(container, auth),
	)

	app.Post("/auth/reset/send",
		middleware.ValidateBody[schemas.ResetPasswordRequest](),
		ReverseProxy(container, auth))

	app.Post("/auth/reset/verify",
		middleware.ValidateBody[schemas.ResetCodeVerifyRequest](),
		ReverseProxy(container, auth))

	app.Post("/auth/reset/final",
		middleware.ValidateBody[schemas.ResetPasswordFinalRequest](),
		ReverseProxy(container, auth))

	// --- AUTH SERVICE (Zabezpieczone) ---
	app.Post("/auth/register-device",
		middleware.ValidateBody[schemas.RegisterDeviceRequest](),
		ReverseProxySecure(container, auth),
	)
	app.Post("/auth/logout",
		middleware.ValidateBody[schemas.RefreshTokenRequest](),
		ReverseProxySecure(container, auth))

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
