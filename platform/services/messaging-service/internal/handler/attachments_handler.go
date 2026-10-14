package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/service"
)

type AttachmentHandler struct {
	service service.AttachmentService
}

func NewAttachmentHandler(s service.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{service: s}
}

func (h *AttachmentHandler) PrepareUpload(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var req model.AttachmentUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	resp, err := h.service.PrepareUpload(ctx, *rc.UserID, req)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *AttachmentHandler) FinalizeUpload(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var req model.AttachmentFinalizeRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	attachment, err := h.service.FinalizeUpload(ctx, *rc.UserID, req)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(attachment)
}

func (h *AttachmentHandler) GetAttachment(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidParams)
	}
	attachment, err := h.service.GetAttachment(ctx, *rc.UserID, id)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(attachment)
}

func (h *AttachmentHandler) DeleteAttachment(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidParams)
	}
	if err := h.service.DeleteAttachment(ctx, *rc.UserID, id); err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
