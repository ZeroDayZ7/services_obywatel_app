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
	logger := shared.GetLogger()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		logger.Warn("[CONTACTS-06] AUTH: user_id missing for contact list request",
			"request_id", requestID,
			"method", c.Method(),
			"path", c.Path(),
		)
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	logger.Info("[CONTACTS-05] HANDLER: retrieving contacts",
		"request_id", requestID,
		"method", c.Method(),
		"path", c.Path(),
		"user_id", rc.UserID.String(),
	)

	contacts, err := h.contactsSvc.GetContacts(ctx, *rc.UserID)
	if err != nil {
		logger.Error("[CONTACTS-08] REPOSITORY: failed to load contacts",
			"request_id", requestID,
			"user_id", rc.UserID.String(),
			"error", err,
		)
		return apperr.SendAppError(c, err)
	}

	logger.Info("[CONTACTS-09] RESPONSE: returning contact list",
		"request_id", requestID,
		"user_id", rc.UserID.String(),
		"incoming", countIncoming(contacts, *rc.UserID),
		"outgoing", countOutgoing(contacts, *rc.UserID),
		"accepted", countAccepted(contacts),
		"total", len(contacts),
	)

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
	logger := shared.GetLogger()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		logger.Warn("[CONTACTS-INVITE-02] AUTH: missing user_id for invite request",
			"request_id", requestID,
			"method", c.Method(),
			"path", c.Path(),
		)
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	var req model.SendContactRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn("[CONTACTS-INVITE-03] HTTP: invalid invite payload",
			"request_id", requestID,
			"user_id", rc.UserID.String(),
			"error", err,
		)
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	if *rc.UserID == req.TargetUserID {
		logger.Warn("[CONTACTS-INVITE-04] VALIDATION: self-invite rejected",
			"request_id", requestID,
			"user_id", rc.UserID.String(),
		)
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	logger.Info("[CONTACTS-INVITE-01] REQUEST: sending contact invite",
		"request_id", requestID,
		"user_id", rc.UserID.String(),
		"target_user_id", req.TargetUserID.String(),
	)

	contact, err := h.contactsSvc.SendRequest(ctx, *rc.UserID, req.TargetUserID)
	if err != nil {
		logger.Error("[CONTACTS-INVITE-05] SERVICE: invite failed",
			"request_id", requestID,
			"user_id", rc.UserID.String(),
			"target_user_id", req.TargetUserID.String(),
			"error", err,
		)
		return apperr.SendAppError(c, err)
	}

	logger.Info("[CONTACTS-INVITE-06] SERVICE: invite created",
		"request_id", requestID,
		"user_id", rc.UserID.String(),
		"target_user_id", req.TargetUserID.String(),
		"contact_id", contact.ID.String(),
	)

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
	logger := shared.GetLogger()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		logger.Warn("[CONTACTS-RESPOND-02] AUTH: missing user_id for response",
			"request_id", requestID,
			"method", c.Method(),
			"path", c.Path(),
		)
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		logger.Warn("[CONTACTS-RESPOND-03] VALIDATION: invalid contact id",
			"request_id", requestID,
			"user_id", rc.UserID.String(),
			"contact_id", c.Params("id"),
		)
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	var req model.RespondContactRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn("[CONTACTS-RESPOND-04] HTTP: invalid accept payload",
			"request_id", requestID,
			"user_id", rc.UserID.String(),
			"contact_id", contactID.String(),
			"error", err,
		)
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	logger.Info("[CONTACTS-RESPOND-01] REQUEST: updating contact response",
		"request_id", requestID,
		"user_id", rc.UserID.String(),
		"contact_id", contactID.String(),
		"accept", req.Accept,
	)

	if err := h.contactsSvc.RespondToRequest(ctx, *rc.UserID, contactID, req.Accept); err != nil {
		logger.Error("[CONTACTS-RESPOND-05] SERVICE: response failed",
			"request_id", requestID,
			"user_id", rc.UserID.String(),
			"contact_id", contactID.String(),
			"accept", req.Accept,
			"error", err,
		)
		return apperr.SendAppError(c, err)
	}

	logger.Info("[CONTACTS-RESPOND-06] SERVICE: response applied",
		"request_id", requestID,
		"user_id", rc.UserID.String(),
		"contact_id", contactID.String(),
		"accept", req.Accept,
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

func countIncoming(contacts []model.Contact, userID uuid.UUID) int {
	count := 0
	for _, contact := range contacts {
		if contact.ContactID == userID {
			count++
		}
	}
	return count
}

func countOutgoing(contacts []model.Contact, userID uuid.UUID) int {
	count := 0
	for _, contact := range contacts {
		if contact.OwnerID == userID {
			count++
		}
	}
	return count
}

func countAccepted(contacts []model.Contact) int {
	count := 0
	for _, contact := range contacts {
		if contact.Status == model.ContactStatusAccepted {
			count++
		}
	}
	return count
}
