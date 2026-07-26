package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

func RegisterNotifyRoutes(app *fiber.App, container *di.Container) {
	target := container.Config.Services.Notify

	notify := app.Group("/notifications")
	notify.All("/*", ReverseProxySecure(container, target))
}