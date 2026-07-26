package router

import (
	"github.com/gofiber/fiber/v2"
	pkgRouter "github.com/zerodayz7/platform/pkg/router"
	"github.com/zerodayz7/platform/pkg/router/health"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

func SetupRoutes(app *fiber.App, container *di.Container) {
	services := container.Config.Services

	// 1. Health checks
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

	// 2. Rejestracja modułów tras
	RegisterAuthRoutes(app, container)
	RegisterDocumentRoutes(app, container)
	RegisterUserRoutes(app, container)
	RegisterNotifyRoutes(app, container)

	// 3. Handlery końcowe (Fallback / 404)
	pkgRouter.SetupFallbackHandlers(app)
}
