package router // lub właściwy pakiet dla routera

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

// RegisterWebSocketRoutes registers WebSocket endpoints.
func RegisterWebSocketRoutes(app *fiber.App, container *di.Container) {
	log := shared.GetLogger()

	target := strings.TrimSpace(container.Config.Services.WS)
	if target == "" {
		log.Warn("WebSocket service target is not configured, skipping route registration")
		return
	}

	log.Info("Registering WebSocket routes", "target", target)

	proxy := websocketProxy(container, target)

	app.Get("/ws", proxy)
	app.Get("/ws/*", proxy)
}
