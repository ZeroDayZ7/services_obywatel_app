package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/mapper"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/service"
)

type UserDocumentHandler struct {
	service service.UserDocumentService
}

func NewUserDocumentHandler(s service.UserDocumentService) *UserDocumentHandler {
	return &UserDocumentHandler{service: s}
}

func (h *UserDocumentHandler) CreateDocument(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var payload model.CreateDocumentPayload
	if err := c.BodyParser(&payload); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}
	if payload.UserID == uuid.Nil {
		payload.UserID = *rc.UserID
	}
	if payload.UserID != *rc.UserID && rc.Role != "ADMIN" {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	doc, err := h.service.CreateDocument(ctx, payload)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(mapper.ToDocumentResponse(*doc))
}

func (h *UserDocumentHandler) GetDocumentByID(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	doc, err := h.service.GetDocumentByID(ctx, id)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToDocumentResponse(*doc))
}

func (h *UserDocumentHandler) GetDocumentsByUserID(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	userID, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}
	if userID != *rc.UserID && rc.Role != "ADMIN" {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	docs, err := h.service.GetDocumentsByUserID(ctx, userID)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToDocumentResponses(docs))
}

func (h *UserDocumentHandler) GetDocumentsMe(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	docs, err := h.service.GetDocumentsByUserID(ctx, *rc.UserID)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(mapper.ToDocumentResponses(docs))
}

func (h *UserDocumentHandler) GetDocumentPDF(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	pdf, contentType, err := h.service.GetDocumentPDF(ctx, id)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.Type(contentType).Send(pdf)
}
