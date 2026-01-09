package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
)

// NewDocsApp tworzy lekką instancję Fiber dla serwisu Docs / Citizen
func NewDocsApp() *fiber.App {
	cfg := AppConfig.Server

	app := fiber.New(fiber.Config{
		ServerHeader:          cfg.ServerHeader,
		BodyLimit:             cfg.BodyLimitMB * 1024 * 1024,
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		IdleTimeout:           cfg.IdleTimeout,
		DisableStartupMessage: true,
		EnableIPValidation:    true,
		TrustedProxies:        []string{"127.0.0.1", "::1"},
		ErrorHandler:          server.ErrorHandler(),
	})

	// Middleware podstawowe
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(shared.NewLimiter("global"))
	app.Use(shared.RequestLoggerMiddleware())

	return app
}
