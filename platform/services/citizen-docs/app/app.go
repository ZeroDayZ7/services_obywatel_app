package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	// pkgMiddleware "github.com/zerodayz7/platform/pkg/middleware"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/di"
)

func NewDocsApp(container *di.Container) *fiber.App {
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

	app.Use(requestid.New())
	app.Use(recover.New())

	app.Use(shared.GetLimiter(shared.LimitGlobal, nil))
	app.Use(shared.RequestLoggerMiddleware())

	// hmacSecret := []byte(container.Config.Internal.HMACSecret)
	// app.Use(pkgMiddleware.InternalAuthMiddleware(hmacSecret))

	return app
}
