package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/permissions"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	gwMiddleware "github.com/zerodayz7/platform/services/gateway/internal/middleware"
)

const ServiceDocuments = "citizen-docs-service"

func RegisterDocumentRoutes(app *fiber.App, container *di.Container) {
	target := container.Config.Services.Documents

	docs := app.Group("/documents")

	// Zwykły obywatel (read / write własnych)
	docs.Get("/me", gwMiddleware.RequirePermissions(permissions.DocumentsRead), ReverseProxySecure(container, ServiceDocuments, target))
	docs.Post("/", gwMiddleware.RequirePermissions(permissions.DocumentsWrite), ReverseProxySecure(container, ServiceDocuments, target))

	// Endpointy specjalne (np. dla Policji / Urzędnika)
	police := app.Group("/police/evidence")

	// Przykład: Wymagane jedno konkretne uprawnienie policyjne
	police.Get("/cases", gwMiddleware.RequirePermissions("police.cases.read"), ReverseProxySecure(container, ServiceDocuments, target))

	// Przykład: Wymagane DWA uprawnienia jednocześnie (np. do odczytu wrażliwych dowodów)
	police.Get("/sensitive-vault", gwMiddleware.RequirePermissions("police.cases.read", "police.vault.classified"), ReverseProxySecure(container, ServiceDocuments, target))
}
