package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
)

func NewAuthApp() *fiber.App {
	app := fiber.New(fiber.Config{
		TrustedProxies:        []string{"127.0.0.1", "::1"},
		BodyLimit:             AppConfig.Server.BodyLimitMB * 1024 * 1024,
		ReadTimeout:           AppConfig.Server.ReadTimeout,
		WriteTimeout:          AppConfig.Server.WriteTimeout,
		IdleTimeout:           AppConfig.Server.IdleTimeout,
		DisableStartupMessage: true,
		EnableIPValidation:    true,
		ServerHeader:          AppConfig.Server.ServerHeader,
		ErrorHandler:          server.ErrorHandler(),
	})

	// Middleware
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(shared.FiberLoggerMiddleware())
	app.Use(shared.NewLimiter("global"))
	app.Use(shared.RequestLoggerMiddleware())

	return app
}
