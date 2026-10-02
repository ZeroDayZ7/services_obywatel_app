package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/zerodayz7/platform/pkg/middleware"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/di"
)

func NewDocsApp(container *di.Container) *fiber.App {
	log := shared.GetLogger()
	cfg := container.Config.Server
	log.Info("[app.NewDocsApp] 1. Initializing Fiber application")

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

	log.Info("[app.NewDocsApp] 2. Registering middleware: requestid + recovery + rate-limiter + request logger")
	app.Use(requestid.New())
	app.Use(recover.New())

	app.Use(shared.GetLimiter(shared.LimitGlobal, nil))
	// Structured HTTP request logging
	app.Use(shared.RequestLoggerMiddleware())
	app.Use(middleware.InternalAuthMiddleware(container.KeyStore))
	log.Info("[app.NewDocsApp] 3. Middleware registration complete")

	return app
}
