package config

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	_ "github.com/gofiber/storage/mysql/v2"
	"github.com/zerodayz7/http-server/internal/middleware"
)

func NewFiberApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"127.0.0.1", "::1"},
		BodyLimit:               2 * 1024 * 1024,
		ReadTimeout:             10 * time.Second,
		WriteTimeout:            10 * time.Second,
		IdleTimeout:             30 * time.Second,
		DisableStartupMessage:   true,
		EnableIPValidation:      true,
		ServerHeader:            "HTTP-Server/ZeroDayZ7",
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

	return app
}
