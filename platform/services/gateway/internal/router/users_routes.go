package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	gwMiddleware "github.com/zerodayz7/platform/services/gateway/internal/middleware"
)

func RegisterUserRoutes(app *fiber.App, container *di.Container) {
	target := container.Config.Services.Users

	users := app.Group("/users", gwMiddleware.RBACRequired("users.manage"))
	users.All("/*", ReverseProxySecure(container, target))
}