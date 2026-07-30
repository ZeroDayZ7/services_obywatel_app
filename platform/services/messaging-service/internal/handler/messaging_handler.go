package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/service"
)

type MessagingHandler struct {
	service service.MessagingService
}

func NewMessagingHandler(s service.MessagingService) *MessagingHandler {
	return &MessagingHandler{service: s}
}

// // #region SyncDelta
func (h *MessagingHandler) SyncDelta(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	log := shared.GetLogger()
	rc := reqctx.MustFromFiber(c)

	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var req model.SyncDeltaRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	resp, err := h.service.GetDeltaSync(ctx, *rc.UserID, req)
	if err != nil {
		log.ErrorObj("Failed to execute sync delta", map[string]any{
			"user_id": rc.UserID.String(),
			"error":   err.Error(),
		})
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// // #endregion

// // #region ProcessOutbox
func (h *MessagingHandler) ProcessOutbox(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var req model.OutboxBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	resp, err := h.service.ProcessOutbox(ctx, *rc.UserID, req)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// #endregion

// #region SendMessage
func (h *MessagingHandler) SendMessage(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var msg model.Message
	if err := c.BodyParser(&msg); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	conversationIDStr := c.Params("id")
	if conversationIDStr != "" {
		convID, err := uuid.Parse(conversationIDStr)
		if err != nil {
			return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
		}
		msg.ConversationID = convID
	}

	if err := h.service.SendMessage(ctx, *rc.UserID, &msg); err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(msg)
}

// #endregion

// #region GetContacts
func (h *MessagingHandler) GetContacts(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	sinceVersionStr := c.Query("since_version", "0")
	sinceVersion, err := strconv.ParseUint(sinceVersionStr, 10, 64)
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	contacts, err := h.service.GetContacts(ctx, *rc.UserID, sinceVersion)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"contacts": contacts})
}

// #endregion

// #region GetConversations
func (h *MessagingHandler) GetConversations(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	conversations, err := h.service.GetConversations(ctx, *rc.UserID)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(conversations)
}

// #endregion

// #region CreateConversation
func (h *MessagingHandler) CreateConversation(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var req model.CreateConversationRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	conv, err := h.service.CreateConversation(ctx, *rc.UserID, req)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(conv)
}

// #endregion

// #region GetConversationByID
func (h *MessagingHandler) GetConversationByID(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	conversationID := c.Params("id")
	conv, err := h.service.GetConversationByID(ctx, *rc.UserID, conversationID)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(conv)
}

// #endregion

// #region GetMessages
func (h *MessagingHandler) GetMessages(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	conversationID := c.Params("id")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	messages, err := h.service.GetMessages(ctx, *rc.UserID, conversationID, limit, offset)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(messages)
}

// #endregion

// #region MarkAsRead
func (h *MessagingHandler) MarkAsRead(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	conversationID := c.Params("id")
	if err := h.service.MarkAsRead(ctx, *rc.UserID, conversationID); err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// #endregion

// #region UploadDeviceKeys
func (h *MessagingHandler) UploadDeviceKeys(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var req model.UploadDeviceKeysRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	if err := h.service.UploadDeviceKeys(ctx, *rc.UserID, req); err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "uploaded"})
}

// #endregion

// #region GetUserPreKeys
func (h *MessagingHandler) GetUserPreKeys(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	targetUserID := c.Params("userId")
	keys, err := h.service.GetUserPreKeys(ctx, targetUserID)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(keys)
}

// #endregion
