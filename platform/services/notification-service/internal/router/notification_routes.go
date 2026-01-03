package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared" // Import Twojego shared
	"github.com/zerodayz7/platform/services/notification-service/internal/handler"
)

func SetupNotificationRoutes(app *fiber.App, notificationH *handler.NotificationHandler) {
	notifications := app.Group("/notifications")
	notifications.Use(shared.NewLimiter("notifications"))

	notifications.Get("/", notificationH.ListMyNotifications)
	notifications.Post("/send", notificationH.SendNotification)

	// Czytanie
	notifications.Patch("/:id/read", notificationH.MarkAsRead)
	notifications.Patch("/read-all", notificationH.MarkAllAsRead)

	// Usuwanie
	notifications.Patch("/:id/trash", notificationH.MoveToTrash)
	notifications.Delete("/trash", notificationH.ClearTrash)

	notifications.Patch("/:id/restore", notificationH.RestoreFromTrash)
	notifications.Delete("/:id", notificationH.DeletePermanently)
}
