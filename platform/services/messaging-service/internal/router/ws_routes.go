package router

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/constants"
	"github.com/zerodayz7/platform/services/messaging-service/internal/di"
	internalWs "github.com/zerodayz7/platform/services/messaging-service/internal/websocket"
)

func SetupWsRoutes(app *fiber.App, container *di.Container) {
	app.Get("/ws/messaging", websocket.New(func(c *websocket.Conn) {
		userID := c.Headers(constants.HeaderUserID)
		if userID == "" {
			userID = c.Query("userId")
		}

		if userID == "" {
			_ = c.Close()
			return
		}

		client := internalWs.NewClient(container.WsHub, c, userID)
		container.WsHub.RegisterClient(client)

		go client.WritePump()
		client.ReadPump()
	}))
}
