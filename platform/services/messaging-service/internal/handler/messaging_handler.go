package handler

import (
	"context"
	"strconv"
	"time"

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

// Sync Delta dla czatów i kontaktów (Offline-First)
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

// SendMessage - wysyłanie zaszyfrowanej wiadomości E2EE
func (h *MessagingHandler) SendMessage(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	// TODO: Dalsze parsowanie i obsługa wysyłki
	_ = ctx
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "queued"})
}

// GetContacts - pobranie listy kontaktów użytkownika
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

	_ = ctx
	_ = sinceVersion
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"contacts": []any{}})
}
