package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	gwMiddleware "github.com/zerodayz7/platform/services/gateway/internal/middleware"
)

const ServiceMessaging = "messaging-service"

// RegisterMessagingRoutes podłącza grupy i endpointy dla kontaktów, konwersacji, wiadomości, sync i E2EE z kontrolą dostępu RBAC.
func RegisterMessagingRoutes(app *fiber.App, container *di.Container) {
	target := container.Config.Services.Messaging

	// #region CONTACTS
	contacts := app.Group("/contacts")
	contacts.Get("", gwMiddleware.RequirePermissions("contacts.read"), ReverseProxySecure(container, ServiceMessaging, target))
	contacts.Post("/request", gwMiddleware.RequirePermissions("contacts.write"), ReverseProxySecure(container, ServiceMessaging, target))
	contacts.Put("/request/:id/respond", gwMiddleware.RequirePermissions("contacts.write"), ReverseProxySecure(container, ServiceMessaging, target))
	// #endregion

	// #region CONVERSATIONS & MESSAGES
	convs := app.Group("/conversations")
	convs.Get("", gwMiddleware.RequirePermissions("messages.read"), ReverseProxySecure(container, ServiceMessaging, target))
	convs.Post("", gwMiddleware.RequirePermissions("messages.write"), ReverseProxySecure(container, ServiceMessaging, target))
	convs.Get("/:id", gwMiddleware.RequirePermissions("messages.read"), ReverseProxySecure(container, ServiceMessaging, target))
	convs.Get("/:id/messages", gwMiddleware.RequirePermissions("messages.read"), ReverseProxySecure(container, ServiceMessaging, target))
	convs.Post("/:id/messages", gwMiddleware.RequirePermissions("messages.write"), ReverseProxySecure(container, ServiceMessaging, target))
	convs.Post("/:id/read", gwMiddleware.RequirePermissions("messages.write"), ReverseProxySecure(container, ServiceMessaging, target))
	// #endregion

	// #region DELTA SYNC & OUTBOX
	sync := app.Group("/sync")
	sync.Get("/delta", gwMiddleware.RequirePermissions("messages.read"), ReverseProxySecure(container, ServiceMessaging, target))
	sync.Post("/outbox", gwMiddleware.RequirePermissions("messages.write"), ReverseProxySecure(container, ServiceMessaging, target))
	// #endregion

	// #region E2EE CRYPTO KEYS EXCHANGE
	crypto := app.Group("/crypto")
	crypto.Post("/keys/device", gwMiddleware.RequirePermissions("messages.write"), ReverseProxySecure(container, ServiceMessaging, target))
	crypto.Get("/keys/prekeys/:userId", gwMiddleware.RequirePermissions("messages.read"), ReverseProxySecure(container, ServiceMessaging, target))
	// #endregion
}
