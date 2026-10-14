package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	// gwMiddleware "github.com/zerodayz7/platform/services/gateway/internal/middleware"
)

const ServiceMessaging = "messaging-service"

// RegisterMessagingRoutes podłącza grupy i endpointy dla kontaktów, konwersacji, wiadomości, sync, E2EE, załączników oraz prywatnej przestrzeni użytkownika.
func RegisterMessagingRoutes(app *fiber.App, container *di.Container) {
	target := container.Config.Services.Messaging

	// --- CONTACTS ---
	contacts := app.Group("/contacts")
	contacts.Get("",
		// gwMiddleware.RequirePermissions("contacts.read"),
		ReverseProxySecure(container, ServiceMessaging, target))
	contacts.Post("/request",
		// gwMiddleware.RequirePermissions("contacts.write"),
		ReverseProxySecure(container, ServiceMessaging, target))
	contacts.Put("/request/:id/respond",
		// gwMiddleware.RequirePermissions("contacts.write"),
		ReverseProxySecure(container, ServiceMessaging, target))

	// --- CONVERSATIONS & MESSAGES ---
	convs := app.Group("/conversations")
	convs.Get("",
		// gwMiddleware.RequirePermissions("messages.read"),
		ReverseProxySecure(container, ServiceMessaging, target))
	convs.Post("",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))
	convs.Get("/:id",
		// gwMiddleware.RequirePermissions("messages.read"),
		ReverseProxySecure(container, ServiceMessaging, target))
	convs.Get("/:id/messages",
		// gwMiddleware.RequirePermissions("messages.read"),
		ReverseProxySecure(container, ServiceMessaging, target))
	convs.Post("/:id/messages",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))
	convs.Post("/:id/read",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))

	// --- PRIVATE USER SPACE ---
	personalSpace := app.Group("/personal-space")
	personalSpace.Get("",
		// gwMiddleware.RequirePermissions("messages.read"),
		ReverseProxySecure(container, ServiceMessaging, target))
	personalSpace.Get("/items",
		// gwMiddleware.RequirePermissions("messages.read"),
		ReverseProxySecure(container, ServiceMessaging, target))
	personalSpace.Post("/items",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))
	personalSpace.Get("/items/:id",
		// gwMiddleware.RequirePermissions("messages.read"),
		ReverseProxySecure(container, ServiceMessaging, target))
	personalSpace.Put("/items/:id",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))
	personalSpace.Delete("/items/:id",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))

	// --- DELTA SYNC & OUTBOX ---
	sync := app.Group("/sync")
	sync.Get("/delta",
		// gwMiddleware.RequirePermissions("messages.read"),
		ReverseProxySecure(container, ServiceMessaging, target))
	sync.Post("/delta",
		// gwMiddleware.RequirePermissions("messages.read"),
		ReverseProxySecure(container, ServiceMessaging, target))
	sync.Post("/outbox",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))

	// --- MESSAGING ACTIVATION & ONBOARDING ---
	messagingActivation := app.Group("/messaging")
	messagingActivation.Get("/terms",
		ReverseProxySecure(container, ServiceMessaging, target))
	messagingActivation.Get("/activation",
		ReverseProxySecure(container, ServiceMessaging, target))
	messagingActivation.Post("/activation",
		ReverseProxySecure(container, ServiceMessaging, target))
	messagingActivation.Post("/activation/consent",
		ReverseProxySecure(container, ServiceMessaging, target))

	// --- E2EE CRYPTO KEYS ---
	crypto := app.Group("/crypto")
	crypto.Post("/keys/device",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))
	crypto.Get("/keys/prekeys/:userId",
		// gwMiddleware.RequirePermissions("messages.read"),
		ReverseProxySecure(container, ServiceMessaging, target))

	e2ee := app.Group("/api/v1/e2ee")
	e2ee.Post("/keys/register",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))
	e2ee.Get("/keys/bundle/:userId",
		// gwMiddleware.RequirePermissions("messages.read"),
		ReverseProxySecure(container, ServiceMessaging, target))

	// --- ATTACHMENTS (S3 / MinIO Proxy) ---
	attachments := app.Group("/attachments")
	attachments.Post("/upload",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))
	attachments.Post("/finalize",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))
	attachments.Get("/:id",
		// gwMiddleware.RequirePermissions("messages.read"),
		ReverseProxySecure(container, ServiceMessaging, target))
	attachments.Delete("/:id",
		// gwMiddleware.RequirePermissions("messages.write"),
		ReverseProxySecure(container, ServiceMessaging, target))
}
