package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/service"
)

type ContactsHandler struct {
	contactsSvc service.ContactsService
}

func NewContactsHandler(s service.ContactsService) *ContactsHandler {
	return &ContactsHandler{contactsSvc: s}
}

// GET /contacts - pobranie listy kontaktów użytkownika
func (h *ContactsHandler) GetContacts(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	requestID := c.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	contacts, err := h.contactsSvc.GetContacts(ctx, *rc.UserID)
	if err != nil {
		log := shared.GetLogger()
		log.Error("[CONTACTS-ERROR] failed to load contacts",
			"request_id", requestID,
			"user_id", rc.UserID.String(),
			"error", err,
		)
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"contacts": contacts})
}

// POST /contacts/request - wysłanie zaproszenia do kontaktów
func (h *ContactsHandler) RequestContact(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	requestID := c.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var req model.SendContactRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	if *rc.UserID == req.TargetUserID {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	contact, err := h.contactsSvc.SendRequest(ctx, *rc.UserID, req.TargetUserID)
	if err != nil {
		log := shared.GetLogger()
		log.Error("[CONTACTS-INVITE-ERROR] invite failed",
			"request_id", requestID,
			"user_id", rc.UserID.String(),
			"target_user_id", req.TargetUserID.String(),
			"error", err,
		)
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(contact)
}

// PUT /contacts/request/:id/respond - akceptacja lub odrzucenie zaproszenia
func (h *ContactsHandler) RespondToRequest(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	requestID := c.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	paramID := c.Params("id")
	contactID, err := uuid.Parse(paramID)
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	var req model.RespondContactRequest
	if err := c.BodyParser(&req); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	if err := h.contactsSvc.RespondToRequest(ctx, *rc.UserID, contactID, req.Accept); err != nil {
		log := shared.GetLogger()
		log.Error("[CONTACTS-RESPOND-ERROR] response failed",
			"request_id", requestID,
			"user_id", rc.UserID.String(),
			"contact_id", contactID.String(),
			"accept", req.Accept,
			"error", err,
		)
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// DELETE /contacts/request/:id - anulowanie własnego oczekującego zaproszenia
func (h *ContactsHandler) CancelRequest(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	requestID := c.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	paramID := c.Params("id")
	contactID, err := uuid.Parse(paramID)
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	if err := h.contactsSvc.CancelRequest(ctx, *rc.UserID, contactID); err != nil {
		log := shared.GetLogger()
		log.Error("[CONTACTS-CANCEL-ERROR] cancel failed",
			"request_id", requestID,
			"user_id", rc.UserID.String(),
			"contact_id", contactID.String(),
			"error", err,
		)
		return apperr.SendAppError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "cancelled"})
}
