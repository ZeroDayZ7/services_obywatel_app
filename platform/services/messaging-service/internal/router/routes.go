package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/messaging-service/internal/di"
)

func SetupMessagingRoutes(app *fiber.App, container *di.Container) {
	msgH := container.MessagingHandler
	contactsH := container.ContactsHandler

	SetupHealthRoutes(app)
	SetupWsRoutes(app, container)

	api := app.Group("/")

	// --- CONTACTS ---
	contacts := api.Group("/contacts")
	contacts.Get("", contactsH.GetContacts)
	contacts.Post("/request", contactsH.RequestContact)
	contacts.Put("/request/:id/respond", contactsH.RespondToRequest)

	// --- CONVERSATIONS & MESSAGES ---
	convs := api.Group("/conversations")
	convs.Get("", msgH.GetConversations)
	convs.Post("", msgH.CreateConversation)
	convs.Get("/:id", msgH.GetConversationByID)
	convs.Get("/:id/messages", msgH.GetMessages)
	convs.Post("/:id/messages", msgH.SendMessage)
	convs.Post("/:id/read", msgH.MarkAsRead)

	// --- DELTA SYNC & OUTBOX ---
	sync := api.Group("/sync")
	sync.Get("/delta", msgH.SyncDelta)
	sync.Post("/outbox", msgH.ProcessOutbox)

	// --- MESSAGING ACTIVATION & ONBOARDING ---
	messagingActivation := api.Group("/messaging")
	messagingActivation.Get("/activation", msgH.GetActivationStatus)
	messagingActivation.Post("/activation", msgH.ActivateMessaging)
	messagingActivation.Post("/activation/consent", msgH.AcceptTerms)

	// --- E2EE CRYPTO KEYS ---
	crypto := api.Group("/crypto")
	crypto.Post("/keys/device", msgH.UploadDeviceKeys)
	crypto.Get("/keys/prekeys/:userId", msgH.GetUserPreKeys)

	v1 := app.Group("/api/v1")
	e2ee := v1.Group("/e2ee")
	e2ee.Post("/keys/register", msgH.UploadDeviceKeys)
	e2ee.Get("/keys/bundle/:userId", msgH.GetKeyBundle)

	SetupFallbackHandlers(app)
}
