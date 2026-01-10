package router

import (
	"github.com/gofiber/fiber/v2"
	pkgRouter "github.com/zerodayz7/platform/pkg/router"
	"github.com/zerodayz7/platform/pkg/router/health"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/internal/di" // Import lokalnego DI!
)

// SetupRoutes konfiguruje routing dla mikroserwisu powiadomień.
func SetupRoutes(app *fiber.App, container *di.Container) {
	// 1. Inicjalizacja handlera z lokalnego kontenera tego serwisu
	h := container.Handlers.NotificationHandler

	// 2. Automatyczne Health Checks (spójne z resztą platformy)
	checker := &health.Checker{
		Service: "notification-service",
		Version: container.Config.Server.AppVersion,
	}
	health.RegisterRoutes(app, checker)

	// 3. Grupa powiadomień
	notifications := app.Group("/notifications")
	{
		// Limiter specyficzny dla powiadomień
		notifications.Use(shared.NewLimiter("notifications"))

		notifications.Get("/", h.ListMyNotifications)
		notifications.Post("/send", h.SendNotification)

		// Obsługa statusów (odczyt)
		notifications.Patch("/:id/read", h.MarkAsRead)
		notifications.Patch("/read-all", h.MarkAllAsRead)

		// Zarządzanie koszem i usuwanie
		notifications.Patch("/:id/trash", h.MoveToTrash)
		notifications.Delete("/trash", h.ClearTrash)
		notifications.Patch("/:id/restore", h.RestoreFromTrash)
		notifications.Delete("/:id", h.DeletePermanently)
	}

	// 4. Globalny Fallback z pkg (404, favicon itp.)
	pkgRouter.SetupFallbackHandlers(app)
}
