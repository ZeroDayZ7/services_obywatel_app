package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/notification-service/internal/service"
)

// NotificationHandler obsługuje endpointy powiadomień
type NotificationHandler struct {
	service *service.NotificationService
}

// NewNotificationHandler tworzy handler
func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: svc}
}

// SendNotification endpoint POST /notifications/send
func (h *NotificationHandler) SendNotification(c *fiber.Ctx) error {
	type Request struct {
		UserID  uint   `json:"user_id"`
		Title   string `json:"title"`
		Message string `json:"message"`
		Type    string `json:"type"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.service.Send(req.UserID, req.Title, req.Message, req.Type); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to send notification"})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

// ListNotifications endpoint GET /notifications/user/:id
func (h *NotificationHandler) ListNotifications(c *fiber.Ctx) error {
	userIDStr := c.Params("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	notifications, err := h.service.ListByUser(uint(userID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch notifications"})
	}

	return c.JSON(notifications)
}
