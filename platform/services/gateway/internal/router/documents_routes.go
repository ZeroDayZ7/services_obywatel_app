package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	gwMiddleware "github.com/zerodayz7/platform/services/gateway/internal/middleware"
)

const ServiceDocuments = "citizen-docs-service"

func RegisterDocumentRoutes(app *fiber.App, container *di.Container) {
	target := container.Config.Services.Documents

	docs := app.Group("/documents")

	// 1:1 odzwierciedlenie tras z mikroserwisu citizen-docs
	docs.Post("/", gwMiddleware.RBACRequired("documents.write"), ReverseProxySecure(container, ServiceDocuments, target))
	docs.Get("/me", gwMiddleware.RBACRequired("documents.read"), ReverseProxySecure(container, ServiceDocuments, target))
}