package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/di"
)

// NewDocsApp tworzy instancję Fiber przyjmując kontener DI
func NewDocsApp(container *di.Container) *fiber.App {
	// Pobieramy konfigurację serwera z kontenera
	cfg := container.Config.Server

	app := fiber.New(fiber.Config{
		AppName:                 cfg.AppName,
		ServerHeader:            cfg.ServerHeader,
		Prefork:                 cfg.Prefork,
		CaseSensitive:           cfg.CaseSensitive,
		StrictRouting:           cfg.StrictRouting,
		BodyLimit:               cfg.BodyLimitMB * 1024 * 1024,
		ReadTimeout:             cfg.ReadTimeout,
		WriteTimeout:            cfg.WriteTimeout,
		IdleTimeout:             cfg.IdleTimeout,
		DisableStartupMessage:   true,
		EnableIPValidation:      true,
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"127.0.0.1", "::1"},
		ErrorHandler:            server.ErrorHandler(),
	})

	// Middleware podstawowe
	app.Use(requestid.New())
	app.Use(recover.New())

	// Limiter i Logger z pkg/shared
	app.Use(shared.NewLimiter("global", nil))
	app.Use(shared.RequestLoggerMiddleware())

	// Jeśli potrzebujesz autoryzacji wewnętrznej (HMAC) jak w auth-service:
	// app.Use(middleware.InternalAuthMiddleware(container.Config.Internal.HMACSecret))

	return app
}
