package handler

import (
	"context"
	"strconv"
	"strings"
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

	rawBody := string(c.Body())
	shared.GetLogger().Info("[OUTBOX_DEBUG] raw request body",
		"user_id", rc.UserID.String(),
		"body", rawBody,
	)

	var req model.OutboxBatchRequest
	if err := c.BodyParser(&req); err != nil {
		shared.GetLogger().Error("[OUTBOX_DEBUG] failed to parse outbox JSON",
			"user_id", rc.UserID.String(),
			"body", rawBody,
			"error", err.Error(),
		)
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	shared.GetLogger().Info("[OUTBOX_DEBUG] parsed request",
		"user_id", rc.UserID.String(),
		"message_count", len(req.Messages),
	)
	for i, msg := range req.Messages {
		shared.GetLogger().Info("[OUTBOX_DEBUG] message item",
			"index", i,
			"event_id", msg.EventID,
			"event_type", msg.EventType,
			"conversation_id", msg.ConversationID,
			"sender_device_id", msg.SenderDeviceID,
			"ciphertext_len", len(msg.Ciphertext),
			"type", msg.Type,
			"idempotency_key", msg.IdempotencyKey,
		)
	}

	resp, err := h.service.ProcessOutbox(ctx, *rc.UserID, req)
	if err != nil {
		shared.GetLogger().Error("[OUTBOX_DEBUG] service rejected request",
			"user_id", rc.UserID.String(),
			"error", err.Error(),
		)
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// #endregion

// #region SendMessage
func (h *MessagingHandler) SendMessage(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	logger := shared.GetLogger()
	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var req model.SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("[MESSAGE_REQUEST_PARSE_ERROR] invalid body for message send",
			"user_id", rc.UserID.String(),
			"error", err.Error())
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}
	req.Normalize()

	conversationIDStr := c.Params("id")
	if req.ConversationID == nil && conversationIDStr != "" {
		convID, err := uuid.Parse(conversationIDStr)
		if err != nil {
			logger.Error("[MESSAGE_REQUEST_PARSE_ERROR] invalid conversation id",
				"user_id", rc.UserID.String(),
				"conversation_id", conversationIDStr,
				"error", err.Error())
			return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
		}
		req.ConversationID = &convID
	}
	if req.ConversationID == nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	logger.Info("[MESSAGE_RECEIVE_REQUEST] inbound message request",
		"user_id", rc.UserID.String(),
		"conversation_id", req.ConversationID.String(),
		"sender_device_id", req.SenderDeviceID,
		"ciphertext_len", len(req.Ciphertext),
		"content_len", len(req.Content),
	)

	resolvedSignalType := req.Type
	if resolvedSignalType == 0 {
		resolvedSignalType = req.TypeSnake
	}
	messageType := model.MessageTypeText
	if resolvedSignalType != 0 {
		messageType = model.MessageType(strconv.Itoa(int(resolvedSignalType)))
	}

	msg := &model.Message{
		ConversationID:   *req.ConversationID,
		SenderID:         *rc.UserID,
		SenderDeviceID:   req.SenderDeviceID,
		Type:             messageType,
		EncryptedPayload: req.Ciphertext,
		IdempotencyKey:   strings.TrimSpace(req.IdempotencyKey),
	}
	if req.Content != "" {
		msg.EncryptedPayload = []byte(req.Content)
	}
	if msg.IdempotencyKey == "" {
		msg.IdempotencyKey = uuid.NewString()
	}

	if err := h.service.SendMessage(ctx, *rc.UserID, msg); err != nil {
		logger.Error("[MESSAGE_SEND_FAILED] service rejected message",
			"user_id", rc.UserID.String(),
			"conversation_id", req.ConversationID.String(),
			"error", err.Error())
		return apperr.SendAppError(c, err)
	}

	logger.Info("[MESSAGE_SEND_OK] message accepted by backend",
		"user_id", rc.UserID.String(),
		"conversation_id", req.ConversationID.String(),
		"message_id", msg.ID.String(),
	)

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
	req.Normalize()

	// Log a minimal, non-sensitive trace to help debug missing DB records in production.
	// Do NOT log any key material.
	shared.GetLogger().InfoObj("Uploading device keys", map[string]any{
		"user_id":                 rc.UserID.String(),
		"device_id":               req.DeviceID,
		"one_time_pre_keys_count": len(req.OneTimePreKeys),
		"operation_id":            rc.OperationID,
	})

	if err := h.service.UploadDeviceKeys(ctx, *rc.UserID, req); err != nil {
		shared.GetLogger().ErrorObj("Failed to upload device keys", map[string]any{
			"user_id":      rc.UserID.String(),
			"device_id":    req.DeviceID,
			"error":        err.Error(),
			"operation_id": rc.OperationID,
		})
		return apperr.SendAppError(c, err)
	}

	shared.GetLogger().InfoObj("Device keys uploaded", map[string]any{
		"user_id":                 rc.UserID.String(),
		"device_id":               req.DeviceID,
		"one_time_pre_keys_count": len(req.OneTimePreKeys),
		"operation_id":            rc.OperationID,
	})

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

func (h *MessagingHandler) GetKeyBundle(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	targetUserID := c.Params("userId")
	bundle, err := h.service.GetUserKeyBundle(ctx, targetUserID)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(bundle)
}

func (h *MessagingHandler) GetCurrentTerms(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	terms, err := h.service.GetCurrentTerms(ctx)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(terms)
}

func (h *MessagingHandler) GetActivationStatus(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	resp, err := h.service.GetActivationStatus(ctx, *rc.UserID)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *MessagingHandler) ActivateMessaging(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var req model.ActivateMessagingRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	resp, err := h.service.ActivateMessaging(ctx, *rc.UserID, req)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *MessagingHandler) AcceptTerms(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var req model.AcceptTermsRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	resp, err := h.service.AcceptTerms(ctx, *rc.UserID, req)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// #endregion
