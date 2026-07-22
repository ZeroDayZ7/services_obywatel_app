package router

import (
	"github.com/gofiber/fiber/v2"
	pkgMiddleware "github.com/zerodayz7/platform/pkg/middleware"
	pkgRouter "github.com/zerodayz7/platform/pkg/router"
	"github.com/zerodayz7/platform/pkg/router/health"
	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	gwMiddleware "github.com/zerodayz7/platform/services/gateway/internal/middleware"
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
		ReverseProxy(container, auth),
	)

	app.Post("/auth/verify-device",
		pkgMiddleware.ValidateBody[schemas.VerifyDeviceRequest](),
		ReverseProxy(container, auth),
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

	// --- AUTH SERVICE (Zabezpieczone - Użytkownik Zalogowany) ---
	// ContextBuilder odczytuje token JWT i weryfikuje aktywną sesję w Redisie
	authGroup := app.Group("", gwMiddleware.ContextBuilder(container))

	authGroup.Get("/auth/me", ReverseProxySecure(container, auth))
	authGroup.Post("/auth/register-device",
		pkgMiddleware.ValidateBody[schemas.RegisterDeviceRequest](),
		ReverseProxySecure(container, auth),
	)
	authGroup.Post("/auth/logout",
		pkgMiddleware.ValidateBody[schemas.RefreshTokenRequest](),
		ReverseProxySecure(container, auth),
	)
	authGroup.Get("/user/sessions", ReverseProxySecure(container, auth))
	authGroup.Post("/user/sessions/terminate", ReverseProxySecure(container, auth))

	// --- NOTIFICATIONS (Zabezpieczone) ---
	notify := services.Notify
	authGroup.All("/notifications*", ReverseProxySecure(container, notify))

	// --- DOCUMENTS (Przykład RBAC: wymagana rola/permisja 'documents:read') ---
	documents := services.Documents
	authGroup.All("/documents/*",
		gwMiddleware.RBACRequired("documents:read"),
		ReverseProxySecure(container, documents),
	)

	// --- USERS SERVICE (Przykład RBAC: wymagana rola 'admin' lub permisja 'users:manage') ---
	users := services.Users
	authGroup.All("/users/*",
		gwMiddleware.RBACRequired("users:manage"),
		ReverseProxySecure(container, users),
	)

	// Fallback (404 / 405)
	pkgRouter.SetupFallbackHandlers(app)
}
