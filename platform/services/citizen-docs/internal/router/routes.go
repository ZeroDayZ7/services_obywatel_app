package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/di"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/handler"
)

func SetupDocsRoutes(app *fiber.App, container *di.Container) {
	log := shared.GetLogger()
	h := handler.NewUserDocumentHandler(container.UserDocumentSvc)

	log.Info("[router.SetupDocsRoutes] 1. Registering health and document routes")
	SetupHealthRoutes(app)

	api := app.Group("/api/v1")
	docs := api.Group("/documents")
	users := api.Group("/users")

	docs.Post("", h.CreateDocument)
	docs.Get("/:id", h.GetDocumentByID)
	docs.Get("/:id/pdf", h.GetDocumentPDF)
	docs.Get("/me", h.GetDocumentsMe)

	users.Get("/:user_id/documents", h.GetDocumentsByUserID)

	legacyDocs := app.Group("/documents")
	legacyDocs.Post("", h.CreateDocument)
	legacyDocs.Get("/:id", h.GetDocumentByID)
	legacyDocs.Get("/:id/pdf", h.GetDocumentPDF)
	legacyDocs.Get("/me", h.GetDocumentsMe)

	SetupFallbackHandlers(app)
	log.Info("[router.SetupDocsRoutes] 2. All routes and fallback handlers registered")
}
