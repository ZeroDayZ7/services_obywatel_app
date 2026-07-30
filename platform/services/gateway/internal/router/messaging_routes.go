package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	gwMiddleware "github.com/zerodayz7/platform/services/gateway/internal/middleware"
)

// RegisterMessagingRoutes podłącza grupy i endpointy dla kontaktów, konwersacji, wiadomości, sync i E2EE z kontrolą dostępu RBAC.
func RegisterMessagingRoutes(app *fiber.App, container *di.Container) {
	target := container.Config.Services.Messaging

	// #region CONTACTS
	contacts := app.Group("/contacts")
	contacts.Get("", gwMiddleware.RBACRequired("contacts.read"), ReverseProxySecure(container, target))
	contacts.Post("/request", gwMiddleware.RBACRequired("contacts.write"), ReverseProxySecure(container, target))
	contacts.Put("/request/:id/respond", gwMiddleware.RBACRequired("contacts.write"), ReverseProxySecure(container, target))
	// #endregion

	// #region CONVERSATIONS & MESSAGES
	convs := app.Group("/conversations")
	convs.Get("", gwMiddleware.RBACRequired("messages.read"), ReverseProxySecure(container, target))
	convs.Post("", gwMiddleware.RBACRequired("messages.write"), ReverseProxySecure(container, target))
	convs.Get("/:id", gwMiddleware.RBACRequired("messages.read"), ReverseProxySecure(container, target))
	convs.Get("/:id/messages", gwMiddleware.RBACRequired("messages.read"), ReverseProxySecure(container, target))
	convs.Post("/:id/messages", gwMiddleware.RBACRequired("messages.write"), ReverseProxySecure(container, target))
	convs.Post("/:id/read", gwMiddleware.RBACRequired("messages.write"), ReverseProxySecure(container, target))
	// #endregion

	// #region DELTA SYNC & OUTBOX
	sync := app.Group("/sync")
	sync.Get("/delta", gwMiddleware.RBACRequired("messages.read"), ReverseProxySecure(container, target))
	sync.Post("/outbox", gwMiddleware.RBACRequired("messages.write"), ReverseProxySecure(container, target))
	// #endregion

	// #region E2EE CRYPTO KEYS EXCHANGE
	crypto := app.Group("/crypto")
	crypto.Post("/keys/device", gwMiddleware.RBACRequired("messages.write"), ReverseProxySecure(container, target))
	crypto.Get("/keys/prekeys/:userId", gwMiddleware.RBACRequired("messages.read"), ReverseProxySecure(container, target))
	// #endregion
}
