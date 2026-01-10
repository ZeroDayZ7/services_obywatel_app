package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/internal/di"
)

func NewNotificationApp(container *di.Container) *fiber.App {
	// Pobieramy konfigurację serwera z kontenera
	cfg := container.Config.Server

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

	// Jeśli potrzebujesz InternalAuthMiddleware w powiadomieniach:
	// app.Use(middleware.InternalAuthMiddleware(container.InternalSecret))

	return app
}
