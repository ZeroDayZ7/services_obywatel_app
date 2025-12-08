package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/http-server/internal/handler"
	"github.com/zerodayz7/http-server/internal/service"
)

// SetupDocsRoutes ustawia wszystkie trasy dla mikroserwisu dokumentów
func SetupDocsRoutes(app *fiber.App, userDocService *service.UserDocumentService) {
	h := handler.NewUserDocumentHandler(userDocService)

	SetupHealthRoutes(app) // np. /health

	docs := app.Group("/documents")

	docs.Post("/", h.CreateDocument)
	docs.Get("/user/:id", h.GetDocumentsByUserID)
	// Możesz dodać pozostałe operacje np. Get/:id, Put/:id, Delete/:id
	// docs.Get("/:id", h.GetDocument)
	// docs.Put("/:id", h.UpdateDocument)
	// docs.Delete("/:id", h.DeleteDocument)

	SetupFallbackHandlers(app)
}
