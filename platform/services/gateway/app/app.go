package app

import (
	"fmt"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/config"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	"github.com/zerodayz7/platform/services/gateway/internal/middleware"
)

//#region NewGatewayApp
func NewGatewayApp(container *di.Container) (*fiber.App, error) {
	if container == nil {
		return nil, fmt.Errorf("container is nil")
	}

	if container.Redis == nil {
		return nil, fmt.Errorf("redis connection is required for Gateway to start")
	}

	cfg := container.Config.Server

	cfgFiber := fiber.Config{
		AppName:                 cfg.AppName,
		ServerHeader:            cfg.ServerHeader,
		Prefork:                 cfg.Prefork,
		CaseSensitive:           cfg.CaseSensitive,
		StrictRouting:           cfg.StrictRouting,
		IdleTimeout:             cfg.IdleTimeout,
		ReadTimeout:             cfg.ReadTimeout,
		WriteTimeout:            cfg.WriteTimeout,
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"127.0.0.1", "::1"},
		BodyLimit:               cfg.BodyLimitMB * 1024 * 1024,
		DisableStartupMessage:   true,
		EnableIPValidation:      true,
		ErrorHandler:            server.ErrorHandler(),
	}

	app := fiber.New(cfgFiber)

	app.Use(otelfiber.Middleware())
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(helmet.New(config.HelmetConfig()))
	app.Use(cors.New(config.CorsConfig()))

	storage := container.Redis.AsFiberStorage()

	app.Use(shared.GetLimiter(shared.LimitGlobal, storage))
	app.Use(compress.New(config.CompressConfig()))
	// app.Use(shared.RequestLoggerMiddleware())

	app.Use(config.JWTMiddlewareWithExclusions())
	app.Use(middleware.AuthRedisMiddleware(container.Cache))
	app.Use(middleware.ContextBuilder(container))

	return app, nil
}
