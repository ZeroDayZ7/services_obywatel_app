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

	api := app.Group("/")

	hmacSecret := []byte(container.Config.Internal.HMACSecret)
	api.Use(pkgMiddleware.InternalAuthMiddleware(hmacSecret))

	// --- CONVERSATIONS & MESSAGES ---
	convs := api.Group("/conversations")
	convs.Get("", h.GetConversations)
	convs.Post("", h.CreateConversation)
	convs.Get("/:id", h.GetConversationByID)
	convs.Get("/:id/messages", h.GetMessages)
	convs.Post("/:id/messages", h.SendMessage)
	convs.Post("/:id/read", h.MarkAsRead)

	// --- DELTA SYNC & OUTBOX ---
	sync := api.Group("/sync")
	sync.Get("/delta", h.SyncDelta)
	sync.Post("/outbox", h.ProcessOutbox)

	// --- E2EE CRYPTO KEYS ---
	crypto := api.Group("/crypto")
	crypto.Post("/keys/device", h.UploadDeviceKeys)
	crypto.Get("/keys/prekeys/:userId", h.GetUserPreKeys)

	SetupFallbackHandlers(app)
}
