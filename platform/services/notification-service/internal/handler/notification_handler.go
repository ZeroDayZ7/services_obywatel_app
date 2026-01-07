package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"github.com/zerodayz7/platform/services/notification-service/internal/service"
)

type NotificationHandler struct {
	service *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: svc}
}

// Funkcja pomocnicza, aby nie powtarzać logiki parsowania UserID
func (h *NotificationHandler) parseUserID(c *fiber.Ctx) (uuid.UUID, error) {
	userIDStr := c.Get("X-User-Id")
	if userIDStr == "" {
		userIDStr = c.Get("X-User-ID") // Sprawdzamy obie wersje
	}
	return uuid.Parse(userIDStr)
}

// ListMyNotifications GET /notifications
func (h *NotificationHandler) ListMyNotifications(c *fiber.Ctx) error {
	userID, err := h.parseUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or missing user id"})
	}

	notifications, err := h.service.ListByUser(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch notifications"})
	}
	return c.JSON(notifications)
}

// MarkAsRead PATCH /notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid notification id"})
	}

	userID, err := h.parseUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or missing user id"})
	}

	if err := h.service.MarkRead(id, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to mark as read"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// MarkAllAsRead PATCH /notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userID, err := h.parseUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or missing user id"})
	}

	if err := h.service.MarkAllRead(userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to mark all as read"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// MoveToTrash PATCH /notifications/:id/trash
func (h *NotificationHandler) MoveToTrash(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid notification id"})
	}

	userID, err := h.parseUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or missing user id"})
	}

	if err := h.service.MoveToTrash(id, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to move to trash"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ClearTrash DELETE /notifications/trash
func (h *NotificationHandler) ClearTrash(c *fiber.Ctx) error {
	userID, err := h.parseUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or missing user id"})
	}

	if err := h.service.ClearTrash(userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to clear trash"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// SendNotification POST /notifications/send (Wewnętrzne/Admin)
func (h *NotificationHandler) SendNotification(c *fiber.Ctx) error {
	var req model.Notification
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.service.Send(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to send notification"})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

// RestoreFromTrash PATCH /notifications/:id/restore
func (h *NotificationHandler) RestoreFromTrash(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid notification id"})
	}

	userID, err := h.parseUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or missing user id"})
	}

	if err := h.service.Restore(id, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to restore notification"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// DeletePermanently DELETE /notifications/:id
func (h *NotificationHandler) DeletePermanently(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid notification id"})
	}

	userID, err := h.parseUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or missing user id"})
	}

	if err := h.service.DeletePermanently(id, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete notification permanently"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
