package router

import (
	"github.com/gofiber/fiber/v2"
	pkgRouter "github.com/zerodayz7/platform/pkg/router"
	"github.com/zerodayz7/platform/pkg/router/health"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

//#region SetupRoutes
func SetupRoutes(app *fiber.App, container *di.Container) {
	services := container.Config.Services

	// 1. Health checks
	checker := &health.Checker{
		Redis:   container.Redis.Client,
		Service: "gateway",
		Version: container.Config.Server.AppVersion,
		Upstreams: map[string]string{
			"auth":       services.Auth + "/health",
			"documents":  services.Documents + "/health",
			"notify":     services.Notify + "/health",
			"messaging":  services.Messaging + "/health",
			"identity":   services.Identity + "/health",
			"officerBFF": services.OfficerBFF + "/health",
		},
	}
	health.RegisterRoutes(app, checker)

	// 2. Rejestracja modułów tras
	RegisterAuthRoutes(app, container)
	RegisterDocumentRoutes(app, container)
	RegisterNotifyRoutes(app, container)
	RegisterMessagingRoutes(app, container)
	RegisterOfficerRoutes(app, container)

	RegisterWsRoutes(app, container)

	// 3. Handlery końcowe (Fallback / 404)
	pkgRouter.SetupFallbackHandlers(app)
}
