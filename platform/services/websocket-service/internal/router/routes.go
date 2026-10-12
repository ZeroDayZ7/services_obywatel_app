package router

import (
	fiber "github.com/gofiber/fiber/v3"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/websocket-service/internal/di"
	"github.com/zerodayz7/platform/services/websocket-service/internal/handler"
)

func SetupRoutes(app *fiber.App, container *di.Container) {
	log := shared.GetLogger()
	log.Info("Initializing WebSocket service routes...")

	wsHandler := handler.NewHandler(container.Hub)

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Get("/ws", wsHandler.HandleWS)
	app.Post("/internal/broadcast", wsHandler.HandleInternalBroadcast)

	log.Info("WebSocket service routes successfully registered",
		"endpoints", []string{"/health", "/ws", "/internal/broadcast"},
	)
}
