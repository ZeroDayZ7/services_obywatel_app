package app

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	// "github.com/zerodayz7/platform/pkg/server" // Odkomentuj, jeśli chcesz używać ujednoliconego handlera błędów z platformy
	// "github.com/zerodayz7/platform/pkg/shared" // Odkomentuj, jeśli chcesz dodać globalny limiter
	"github.com/zerodayz7/platform/services/websocket-service/internal/di"
	"github.com/zerodayz7/platform/services/websocket-service/internal/middleware"
)

func NewApp(container *di.Container) *fiber.App {
	cfg := container.Config.Server

	app := fiber.New(fiber.Config{
		AppName:       cfg.AppName,
		ServerHeader:  cfg.ServerHeader,
		CaseSensitive: cfg.CaseSensitive,
		StrictRouting: cfg.StrictRouting,
		IdleTimeout:   cfg.IdleTimeout,
		ReadTimeout:   cfg.ReadTimeout,
		WriteTimeout:  cfg.WriteTimeout,
		BodyLimit:     cfg.BodyLimitMB * 1024 * 1024,

		EnableIPValidation: true,
		ProxyHeader:        fiber.HeaderXForwardedFor,
		TrustProxy:         true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			// Proxies: []string{"127.0.0.1", "::1"},

			// LUB zamiast wpisywać ręcznie, możesz użyć wbudowanej flagi dla localhost:
			Loopback: true,
		},
		// ErrorHandler:         server.ErrorHandler(), // Zbędne/opcjonalne w czystym WS, odkomentuj w razie potrzeby REST

		// --- ZBĘDNE / OPCJONALNE (zakomentowane) ---
		// Prefork: cfg.Prefork, // Zazwyczaj false dla websocketów, żeby nie komplikować sesji w pamięci Hub
	})

	app.Use(requestid.New())
	app.Use(recover.New())

	// Zbędne dla czystego WebSocket (może odcinać ruch), odkomentuj tylko jeśli potrzebujesz rate limitingu na HTTP:
	// app.Use(shared.GetLimiter(shared.LimitGlobal, nil))

	app.Use(middleware.InternalAuthMiddleware(container.KeyStore))

	// Opcjonalne (jeśli chcesz śledzić ID operacji również w websocket-service):
	// app.Use(middleware.OperationIdMiddleware())

	return app
}
