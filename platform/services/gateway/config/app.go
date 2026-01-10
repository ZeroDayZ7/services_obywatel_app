package config

import (
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	"github.com/zerodayz7/platform/services/gateway/internal/middleware"
)

func NewGatewayApp(container *di.Container) *fiber.App {
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
	app.Use(otelfiber.Middleware())
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(helmet.New(HelmetConfig()))
	app.Use(cors.New(CorsConfig()))
	app.Use(shared.NewLimiter("global"))
	app.Use(compress.New(CompressConfig()))
	app.Use(shared.RequestLoggerMiddleware())
	app.Use(JWTMiddlewareWithExclusions())
	app.Use(middleware.AuthRedisMiddleware(container.Redis.Client))
	app.Use(middleware.ContextBuilder())

	return app
}
