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

	app.Post("/auth/2fa-resend",
		pkgMiddleware.ValidateBody[schemas.ResendTwoFARequest](),
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

	app.Post("/auth/unpair-device",
		ReverseProxySecure(container, auth),
	)

	app.Get("/auth/me", ReverseProxySecure(container, auth))
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

	notify := services.Notify
	app.All("/notifications*", ReverseProxySecure(container, notify))

	documents := services.Documents
	app.All("/documents/*",
		gwMiddleware.RBACRequired("documents.read"),
		ReverseProxySecure(container, documents),
	)

	users := services.Users
	app.All("/users/*",
		gwMiddleware.RBACRequired("users.manage"),
		ReverseProxySecure(container, users),
	)

	pkgRouter.SetupFallbackHandlers(app)
}
