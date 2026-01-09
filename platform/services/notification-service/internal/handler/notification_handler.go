package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/utils"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"github.com/zerodayz7/platform/services/notification-service/internal/service"
)

type NotificationHandler struct {
	service *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: svc}
}

// Przykład z pełnym komentarzem dla zrozumienia mechanizmu:
func (h *NotificationHandler) ListMyNotifications(c *fiber.Ctx) error {
	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	// 1. DZIEDZICZYMY z c.Context() (jeśli klient rozłączy, my kończymy pracę)
	// 2. NAKŁADAMY limit czasu (jeśli baza muli powyżej 5s, my kończymy pracę)
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	notifications, err := h.service.ListByUser(ctx, userID)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}
	return c.JSON(notifications)
}

func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	id, err := utils.ParseUUID(c, "id")
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidRequest)
	}

	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	if err := h.service.MarkRead(ctx, id, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	if err := h.service.MarkAllRead(ctx, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *NotificationHandler) MoveToTrash(c *fiber.Ctx) error {
	id, err := utils.ParseUUID(c, "id")
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidRequest)
	}

	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	if err := h.service.MoveToTrash(ctx, id, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *NotificationHandler) ClearTrash(c *fiber.Ctx) error {
	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.Context(), 15*time.Second)
	defer cancel()

	if err := h.service.ClearTrash(ctx, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *NotificationHandler) SendNotification(c *fiber.Ctx) error {
	var req model.Notification
	if err := c.BodyParser(&req); err != nil {
		return errors.SendAppError(c, errors.ErrInvalidJSON)
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.Send(ctx, &req); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

func (h *NotificationHandler) RestoreFromTrash(c *fiber.Ctx) error {
	id, err := utils.ParseUUID(c, "id")
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidRequest)
	}

	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	if err := h.service.Restore(ctx, id, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *NotificationHandler) DeletePermanently(c *fiber.Ctx) error {
	id, err := utils.ParseUUID(c, "id")
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidRequest)
	}

	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.DeletePermanently(ctx, id, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
