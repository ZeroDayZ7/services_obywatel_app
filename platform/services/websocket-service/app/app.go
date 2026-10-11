package app

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/zerodayz7/platform/services/websocket-service/internal/config"
)

func NewApp(cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:       cfg.Server.AppName,
		ServerHeader:  cfg.Server.ServerHeader,
		CaseSensitive: cfg.Server.CaseSensitive,
		StrictRouting: cfg.Server.StrictRouting,
		IdleTimeout:   cfg.Server.IdleTimeout,
		ReadTimeout:   cfg.Server.ReadTimeout,
		WriteTimeout:  cfg.Server.WriteTimeout,
		BodyLimit:     cfg.Server.BodyLimitMB * 1024 * 1024,
	})

	app.Use(recover.New())
	app.Use(requestid.New())

	return app
}
