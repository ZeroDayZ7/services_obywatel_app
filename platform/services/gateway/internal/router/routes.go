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

	// --- PUBLICZNE PROXY ---
	// Przekazujemy 'container' do każdej funkcji
	app.Post("/auth/login", ReverseProxy(container, "http://localhost:8082"))
	app.Post("/auth/2fa-verify", ReverseProxy(container, "http://localhost:8082"))
	app.Post("/auth/refresh", ReverseProxy(container, "http://localhost:8082"))

	// --- ZABEZPIECZONE PROXY (Wymaga JWT) ---
	// Te endpointy automatycznie dodadzą X-User-Id do nagłówka
	app.Post("/auth/register-device", ReverseProxySecure(container, "http://localhost:8082"))
	app.Post("/auth/logout", ReverseProxySecure(container, "http://localhost:8082"))

	// SESJE (do ekranu we Flutterze)
	app.Get("/user/sessions", ReverseProxySecure(container, "http://localhost:8082"))
	app.Post("/user/sessions/terminate", ReverseProxySecure(container, "http://localhost:8082"))

	// DOKUMENTY I INNE
	app.All("/documents/*", ReverseProxySecure(container, "http://localhost:8083"))
	app.All("/users/*", ReverseProxy(container, "http://users-service:3000"))

	SetupFallbackHandlers(app)
}
