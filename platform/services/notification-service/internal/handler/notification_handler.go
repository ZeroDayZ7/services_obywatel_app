package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"github.com/zerodayz7/platform/services/notification-service/internal/service"
)

type NotificationHandler struct {
	service *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: svc}
}

// ListMyNotifications GET /notifications
// Gateway wstrzykuje X-User-ID po dekodowaniu tokena JWT
func (h *NotificationHandler) ListMyNotifications(c *fiber.Ctx) error {
	userIDStr := c.Get("X-User-ID")
	if userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing user id header"})
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id format"})
	}

	notifications, err := h.service.ListByUser(uint(userID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch notifications"})
	}

	return c.JSON(notifications)
}

// MarkAsRead PATCH /notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing notification id"})
	}

	if err := h.service.MarkRead(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to mark as read"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userIDStr := c.Get("X-User-ID")
	userID, _ := strconv.ParseUint(userIDStr, 10, 32)

	if err := h.service.MarkAllRead(uint(userID)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to mark all as read"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// SendNotification POST /notifications/send (Wewnętrzne/Admin)
func (h *NotificationHandler) SendNotification(c *fiber.Ctx) error {
	var req model.Notification // Używamy modelu bezpośrednio lub dedykowanego DTO
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.service.Send(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to send notification"})
	}

	return c.Status(fiber.StatusCreated).JSON(req)
}
