package router

import (
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	SetupHealthRoutes(app)

	// Proxy / redirect do mikroserwisów
	app.All("/auth/*", ReverseProxy("http://localhost:8082"))
	// app.All("/documents/*", ReverseProxy("http://localhost:8083"))
	app.All("/documents/*", ReverseProxyWithUserID("http://localhost:8083"))
	app.All("/users/*", ReverseProxy("http://users-service:3000"))

	SetupFallbackHandlers(app)
}
