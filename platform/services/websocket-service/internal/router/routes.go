package router

import (
	fiber "github.com/gofiber/fiber/v3"
	"github.com/zerodayz7/platform/services/websocket-service/internal/config"
	"github.com/zerodayz7/platform/services/websocket-service/internal/handler"
	"github.com/zerodayz7/platform/services/websocket-service/internal/hub"
)

func SetupRoutes(app *fiber.App, hubManager *hub.Hub, cfg *config.Config) {
	wsHandler := handler.NewHandler(hubManager, cfg.SharedSecret)

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/ws", wsHandler.HandleWS)
	app.Post("/internal/broadcast", wsHandler.HandleInternalBroadcast)
}
