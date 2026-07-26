package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	gwMiddleware "github.com/zerodayz7/platform/services/gateway/internal/middleware"
)

func RegisterDocumentRoutes(app *fiber.App, container *di.Container) {
	target := container.Config.Services.Documents

	// Grupa dokumentów z bazowym RBAC
	docs := app.Group("/documents", gwMiddleware.RBACRequired("documents.read"))

	// Domyślny proxy catch-all dla dokumentów
	docs.All("/*", ReverseProxySecure(container, target))
}
