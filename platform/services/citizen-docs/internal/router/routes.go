package router

import (
	"github.com/gofiber/fiber/v2"
	pkgMiddleware "github.com/zerodayz7/platform/pkg/middleware"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/di"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/handler"
)

func SetupDocsRoutes(app *fiber.App, container *di.Container) {
	h := handler.NewUserDocumentHandler(container.UserDocumentSvc)

	SetupHealthRoutes(app)

	docs := app.Group("/documents")

	// Pobieramy HMAC secret bezpośrednio z kontenera DI
	hmacSecret := []byte(container.Config.Internal.HMACSecret)
	docs.Use(pkgMiddleware.InternalAuthMiddleware(hmacSecret))

	docs.Post("/", h.CreateDocument)
	docs.Get("/me", h.GetDocumentsMe)

	SetupFallbackHandlers(app)
}
