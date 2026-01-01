package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/config"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	"github.com/zerodayz7/platform/services/gateway/internal/router/health"
)

func SetupRoutes(app *fiber.App, container *di.Container) {
	checker := &health.Checker{
		Redis:   container.Redis.Client,
		Service: "gateway",
		Version: config.AppConfig.Server.AppVersion,
		Upstreams: []string{
			"http://auth-service:8082/health",
			"http://citizen-docs:8083/health",
		},
	}
	health.RegisterRoutes(app, checker)

	// Proxy / redirect do mikroserwisów
	app.Post("/auth/login", ReverseProxy("http://localhost:8082"))
	app.Post("/auth/2fa-verify", ReverseProxy("http://localhost:8082"))

	app.Post("/auth/register-device", ReverseProxySecure("http://localhost:8082"))
	app.Post("/auth/logout", ReverseProxySecure("http://localhost:8082"))

	app.All("/documents/*", ReverseProxySecure("http://localhost:8083"))
	app.All("/users/*", ReverseProxy("http://users-service:3000"))

	SetupFallbackHandlers(app)
}
