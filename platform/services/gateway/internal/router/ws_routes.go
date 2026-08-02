package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	gwMiddleware "github.com/zerodayz7/platform/services/gateway/internal/middleware"
)

func RegisterWsRoutes(app *fiber.App, container *di.Container) {
	app.Get(
		"/ws/messaging",
		gwMiddleware.RBACRequired("messaging.access"),
		handleWSProxy(container.Config.Services.Messaging),
	)
}
