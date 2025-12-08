package config

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/zerodayz7/http-server/internal/middleware"
)

func NewFiberApp() *fiber.App {
	app := fiber.New(fiber.Config{
		TrustedProxies:        []string{"127.0.0.1", "::1"},
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           30 * time.Second,
		DisableStartupMessage: true,
		EnableIPValidation:    true,
		ServerHeader:          "Auth-Service/ZeroDayZ7",
	})

	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(FiberLoggerMiddleware())
	app.Use(NewLimiter("global"))
	app.Use(middleware.RequestLoggerMiddleware())

	return app
}
