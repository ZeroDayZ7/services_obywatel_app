package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared" // Import Twojego shared
	"github.com/zerodayz7/platform/services/notification-service/internal/handler"
)

func SetupNotificationRoutes(app *fiber.App, notificationH *handler.NotificationHandler) {
	// Tworzymy grupę i nakładamy limiter na całą grupę
	notifications := app.Group("/notifications")
	notifications.Use(shared.NewLimiter("notifications"))

	// Publiczne/Użytkownika (idzie przez Gateway)
	// Zgodnie z tym co ustaliliśmy, lepiej użyć ListMyNotifications
	notifications.Get("/", notificationH.ListMyNotifications)

	// Endpointy administracyjne / systemowe
	// Jeśli /send jest wywoływane przez inne serwisy, upewnij się,
	// że IP Gatewaya nie zostanie zablokowane (KeyGenerator w shared go wykryje)
	notifications.Post("/send", notificationH.SendNotification)

	// Opcjonalnie: Specyficzny endpoint dla konkretnego powiadomienia
	notifications.Patch("/:id/read", notificationH.MarkAsRead)

	// Dodajemy nową trasę
	notifications.Patch("/read-all", notificationH.MarkAllAsRead)
}
