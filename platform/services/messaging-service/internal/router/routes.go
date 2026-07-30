package router

import (
	"github.com/gofiber/fiber/v2"
	pkgMiddleware "github.com/zerodayz7/platform/pkg/middleware"
	"github.com/zerodayz7/platform/services/messaging-service/internal/di"
)

func SetupMessagingRoutes(app *fiber.App, container *di.Container) {
	h := container.MessagingHandler

	SetupHealthRoutes(app)
	SetupWsRoutes(app, container)

	api := app.Group("/messaging")

	hmacSecret := []byte(container.Config.Internal.HMACSecret)
	api.Use(pkgMiddleware.InternalAuthMiddleware(hmacSecret))

	// Trasy synchronizacji i wiadomości
	api.Post("/sync", h.SyncDelta)
	api.Post("/messages", h.SendMessage)
	api.Get("/contacts", h.GetContacts)

	SetupFallbackHandlers(app)
}
