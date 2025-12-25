package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/notification-service/internal/handler"
)

func SetupNotificationRoutes(app *fiber.App, notificationH *handler.NotificationHandler) {
	notifications := app.Group("/notifications")
	notifications.Post("/send", notificationH.SendNotification)
	notifications.Get("/user/:id", notificationH.ListNotifications)
}
