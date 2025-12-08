package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/zerodayz7/http-server/internal/di"
	"github.com/zerodayz7/http-server/internal/middleware"
)

func NewFiberApp() *fiber.App {

	rdb := NewRedisClient()
	container := di.NewContainer(rdb)

	app := fiber.New(fiber.Config{
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"127.0.0.1", "::1"},
		BodyLimit:               AppConfig.Server.BodyLimitMB * 1024 * 1024,
		ReadTimeout:             AppConfig.Server.ReadTimeout,
		WriteTimeout:            AppConfig.Server.WriteTimeout,
		IdleTimeout:             AppConfig.Server.IdleTimeout,
		DisableStartupMessage:   true,
		EnableIPValidation:      true,
		ServerHeader:            AppConfig.Server.ServerHeader,
	})

	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(FiberLoggerMiddleware())
	app.Use(helmet.New(HelmetConfig()))
	app.Use(cors.New(CorsConfig()))
	app.Use(NewLimiter("global"))
	app.Use(compress.New(CompressConfig()))
	app.Use(middleware.RequestLoggerMiddleware())
	app.Use(JWTMiddlewareWithExclusions())
	app.Use(middleware.AuthRedisMiddleware(container.RedisClient, AppConfig.JWT.AccessSecret))

	return app
}
