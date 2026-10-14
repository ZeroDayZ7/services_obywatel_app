package handler

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/service"
)

type PersonalSpaceHandler struct {
	service service.PersonalSpaceService
}

func NewPersonalSpaceHandler(s service.PersonalSpaceService) *PersonalSpaceHandler {
	return &PersonalSpaceHandler{service: s}
}

func (h *PersonalSpaceHandler) GetOrCreateSpace(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()
	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}
	space, err := h.service.GetOrCreateSpace(ctx, *rc.UserID)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(space)
}

func (h *PersonalSpaceHandler) ListItems(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}
	var sinceVersion uint64
	if value := c.Query("sinceVersion", "0"); value != "" {
		sinceVersion = uint64(c.QueryInt("sinceVersion", 0))
	}
	items, err := h.service.ListItems(ctx, *rc.UserID, sinceVersion)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"items": items})
}

func (h *PersonalSpaceHandler) CreateItem(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}
	var req model.CreatePersonalSpaceItemRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}
	if req.ContentType == "" {
		req.ContentType = model.PersonalSpaceItemTypeText
	}
	if len(req.EncryptedPayload) == 0 {
		return apperr.SendAppError(c, apperr.ErrValidationFailed.WithMeta("detail", "encryptedPayload is required"))
	}
	item, err := h.service.CreateItem(ctx, *rc.UserID, req)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *PersonalSpaceHandler) GetItem(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}
	itemID, err := uuid.Parse(c.Params("id"))
	if err != nil || itemID == uuid.Nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}
	item, err := h.service.GetItem(ctx, *rc.UserID, itemID)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	if item == nil {
		return apperr.SendAppError(c, apperr.ErrNotFound)
	}
	return c.Status(fiber.StatusOK).JSON(item)
}

func (h *PersonalSpaceHandler) UpdateItem(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}
	itemID, err := uuid.Parse(c.Params("id"))
	if err != nil || itemID == uuid.Nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}
	var req model.UpdatePersonalSpaceItemRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}
	req.ItemID = itemID
	item, err := h.service.UpdateItem(ctx, *rc.UserID, req)
	if err != nil {
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(item)
}

func (h *PersonalSpaceHandler) DeleteItem(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}
	itemID, err := uuid.Parse(c.Params("id"))
	if err != nil || itemID == uuid.Nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}
	if err := h.service.DeleteItem(ctx, *rc.UserID, itemID); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return apperr.SendAppError(c, apperr.ErrInternal)
		}
		return apperr.SendAppError(c, err)
	}
	return c.Status(fiber.StatusNoContent).SendString("")
}
