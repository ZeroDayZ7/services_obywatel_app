package router

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

// RegisterWebSocketRoutes registers WebSocket endpoints.
func RegisterWebSocketRoutes(app *fiber.App, container *di.Container) {
	target := strings.TrimSpace(container.Config.Services.WS)
	if target == "" {
		return
	}

	proxy := websocketProxy(container, target)

	app.Get("/ws", proxy)
	app.Get("/ws/*", proxy)
}
