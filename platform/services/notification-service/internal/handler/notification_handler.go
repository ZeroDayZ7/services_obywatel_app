package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/utils"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
)

// #region Interfaces
type NotificationService interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Notification, error)
	MarkRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	MoveToTrash(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ClearTrash(ctx context.Context, userID uuid.UUID) error
	Send(ctx context.Context, n *model.Notification) error
	Restore(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	DeletePermanently(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

// #endregion

type NotificationHandler struct {
	service NotificationService
}

// #region NewNotificationHandler
func NewNotificationHandler(svc NotificationService) *NotificationHandler {
	return &NotificationHandler{service: svc}
}

// #endregion

// #region ListMyNotifications
func (h *NotificationHandler) ListMyNotifications(c *fiber.Ctx) error {
	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	notifications, err := h.service.ListByUser(ctx, userID)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	return c.JSON(notifications)
}

// #endregion

// #region MarkAsRead
func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	id, err := utils.ParseUUID(c, "id")
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidRequest)
	}

	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	if err := h.service.MarkRead(ctx, id, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// #endregion

// #region MarkAllAsRead
func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 10*time.Second)
	defer cancel()

	if err := h.service.MarkAllRead(ctx, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// #endregion

// #region MoveToTrash
func (h *NotificationHandler) MoveToTrash(c *fiber.Ctx) error {
	id, err := utils.ParseUUID(c, "id")
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidRequest)
	}

	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	if err := h.service.MoveToTrash(ctx, id, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// #endregion

// #region ClearTrash
func (h *NotificationHandler) ClearTrash(c *fiber.Ctx) error {
	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 15*time.Second)
	defer cancel()

	if err := h.service.ClearTrash(ctx, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// #endregion

// #region SendNotification
func (h *NotificationHandler) SendNotification(c *fiber.Ctx) error {
	var req model.Notification
	if err := c.BodyParser(&req); err != nil {
		return errors.SendAppError(c, errors.ErrInvalidRequest)
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	if err := h.service.Send(ctx, &req); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	return c.Status(fiber.StatusCreated).JSON(req)
}

// #endregion

// #region RestoreFromTrash
func (h *NotificationHandler) RestoreFromTrash(c *fiber.Ctx) error {
	id, err := utils.ParseUUID(c, "id")
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidRequest)
	}

	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	if err := h.service.Restore(ctx, id, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// #endregion

// #region DeletePermanently
func (h *NotificationHandler) DeletePermanently(c *fiber.Ctx) error {
	id, err := utils.ParseUUID(c, "id")
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidRequest)
	}

	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	if err := h.service.DeletePermanently(ctx, id, userID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// #endregion
