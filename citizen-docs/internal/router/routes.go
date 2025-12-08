package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/http-server/internal/repository"
)

// SetupDocsRoutes ustawia wszystkie trasy dla mikroserwisu dokumentów
func SetupDocsRoutes(app *fiber.App, userDocRepo *repository.UserDocumentRepository) {
	SetupHealthRoutes(app) // np. /health
	SetupDocumentRoutes(app, userDocRepo)
	SetupFallbackHandlers(app)
}

// SetupDocumentRoutes ustawia trasy CRUD dla dokumentów
func SetupDocumentRoutes(app *fiber.App, repo *repository.UserDocumentRepository) {
	docs := app.Group("/documents")

	docs.Post("/", func(c *fiber.Ctx) error {
		// create document
		return fiber.NewError(fiber.StatusNotImplemented, "Create document not implemented")
	})

	docs.Get("/", func(c *fiber.Ctx) error {
		// list all documents
		return fiber.NewError(fiber.StatusNotImplemented, "List documents not implemented")
	})

	docs.Get("/:id", func(c *fiber.Ctx) error {
		// get document by ID
		return fiber.NewError(fiber.StatusNotImplemented, "Get document not implemented")
	})

	docs.Put("/:id", func(c *fiber.Ctx) error {
		// update document by ID
		return fiber.NewError(fiber.StatusNotImplemented, "Update document not implemented")
	})

	docs.Delete("/:id", func(c *fiber.Ctx) error {
		// delete document by ID
		return fiber.NewError(fiber.StatusNotImplemented, "Delete document not implemented")
	})
}
