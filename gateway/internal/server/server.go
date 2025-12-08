// internal/server/server.go
package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/http-server/config"
)

func New(cfg config.Config) *fiber.App {
	return fiber.New(fiber.Config{
		AppName:       cfg.Server.AppName,
		ServerHeader:  cfg.Server.ServerHeader,
		Prefork:       cfg.Server.Prefork,
		CaseSensitive: cfg.Server.CaseSensitive,
		StrictRouting: cfg.Server.StrictRouting,
		BodyLimit:     cfg.Server.BodyLimitMB * 1024 * 1024,
		IdleTimeout:   cfg.Server.IdleTimeout,
		ReadTimeout:   cfg.Server.ReadTimeout,
		WriteTimeout:  cfg.Server.WriteTimeout,
		ErrorHandler:  ErrorHandler(),
	})
}
