package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/zerodayz7/http-server/internal/middleware"
)

func NewFiberApp() *fiber.App {
	_ = NewRedisClient()
	app := fiber.New(fiber.Config{
		TrustedProxies:        []string{"127.0.0.1", "::1"},
		BodyLimit:             AppConfig.Server.BodyLimitMB * 1024 * 1024,
		ReadTimeout:           AppConfig.Server.ReadTimeout,
		WriteTimeout:          AppConfig.Server.WriteTimeout,
		IdleTimeout:           AppConfig.Server.IdleTimeout,
		DisableStartupMessage: true,
		EnableIPValidation:    true,
		ServerHeader:          AppConfig.Server.ServerHeader,
	})

	// Middleware
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(FiberLoggerMiddleware())
	app.Use(NewLimiter("global"))
	app.Use(middleware.RequestLoggerMiddleware())

	return app
}
