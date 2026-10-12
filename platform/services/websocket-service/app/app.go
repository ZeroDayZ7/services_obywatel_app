package app

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/zerodayz7/platform/services/websocket-service/internal/di"
	"github.com/zerodayz7/platform/services/websocket-service/internal/middleware"
)

func NewApp(container *di.Container) *fiber.App {
	cfg := container.Config.Server

	app := fiber.New(fiber.Config{
		AppName:       cfg.AppName,
		ServerHeader:  cfg.ServerHeader,
		CaseSensitive: cfg.CaseSensitive,
		StrictRouting: cfg.StrictRouting,
		IdleTimeout:   cfg.IdleTimeout,
		ReadTimeout:   cfg.ReadTimeout,
		WriteTimeout:  cfg.WriteTimeout,
		BodyLimit:     cfg.BodyLimitMB * 1024 * 1024,
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(middleware.InternalAuthMiddleware(container.KeyStore))

	return app
}
