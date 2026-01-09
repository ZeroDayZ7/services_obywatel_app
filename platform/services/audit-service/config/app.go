package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
)

func NewAuditApp(cfg ServerConfig) *fiber.App {

	cfgFiber := fiber.Config{
		AppName:       cfg.AppName,
		ServerHeader:  cfg.ServerHeader,
		Prefork:       cfg.Prefork,
		CaseSensitive: cfg.CaseSensitive,
		StrictRouting: cfg.StrictRouting,
		IdleTimeout:   cfg.IdleTimeout,
		ReadTimeout:   cfg.ReadTimeout,
		WriteTimeout:  cfg.WriteTimeout,

		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"127.0.0.1", "::1"},
		BodyLimit:               cfg.BodyLimitMB * 1024 * 1024,
		DisableStartupMessage:   true,
		EnableIPValidation:      true,

		ErrorHandler: server.ErrorHandler(),
	}

	app := fiber.New(cfgFiber)

	// Middleware
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(shared.NewLimiter("global"))
	app.Use(shared.RequestLoggerMiddleware())

	return app
}
