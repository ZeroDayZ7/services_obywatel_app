package router

import (
	fiber "github.com/gofiber/fiber/v3"
	"github.com/zerodayz7/platform/services/websocket-service/internal/di"
	"github.com/zerodayz7/platform/services/websocket-service/internal/handler"
)

func SetupRoutes(app *fiber.App, container *di.Container) {
	wsHandler := handler.NewHandler(container.Hub, container.Config.SharedSecret)

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/ws", wsHandler.HandleWS)
	app.Post("/internal/broadcast", wsHandler.HandleInternalBroadcast)
}
