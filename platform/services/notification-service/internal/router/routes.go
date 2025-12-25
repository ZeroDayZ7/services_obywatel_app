package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/notification-service/internal/handler"
)

func SetupRoutes(app *fiber.App, notificationH *handler.NotificationHandler) {
	SetupHealthRoutes(app)
	SetupNotificationRoutes(app, notificationH)
	SetupFallbackHandlers(app)
}
