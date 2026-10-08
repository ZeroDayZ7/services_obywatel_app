package router

import (
	"github.com/gofiber/fiber/v2"

	"github.com/zerodayz7/platform/services/citizen-docs/internal/di"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/handler"
)

func SetupDocsRoutes(app *fiber.App, container *di.Container) {
	h := handler.NewUserDocumentHandler(container.UserDocumentSvc)

	SetupHealthRoutes(app)

	api := app.Group("/api/v1")
	docs := api.Group("/documents")
	users := api.Group("/users")

	docs.Post("", h.CreateDocument)
	docs.Get("/me", h.GetDocumentsMe)
	docs.Get("/:id", h.GetDocumentByID)
	docs.Get("/:id/pdf", h.GetDocumentPDF)

	users.Get("/:user_id/documents", h.GetDocumentsByUserID)

	legacyDocs := app.Group("/documents")
	legacyDocs.Post("", h.CreateDocument)
	legacyDocs.Get("/me", h.GetDocumentsMe)
	legacyDocs.Get("/:id", h.GetDocumentByID)
	legacyDocs.Get("/:id/pdf", h.GetDocumentPDF)

	SetupFallbackHandlers(app)
}
