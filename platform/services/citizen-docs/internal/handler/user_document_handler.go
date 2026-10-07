package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
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

	return c.Status(fiber.StatusCreated).JSON(doc)
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

	return c.Status(fiber.StatusOK).JSON(doc)
}

func (h *UserDocumentHandler) GetDocumentsByUserID(c *fiber.Ctx) error {
	log := shared.GetLogger()
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	log.Info("[UserDocumentHandler.GetDocumentsByUserID] 1. Processing request for /users/:user_id/documents")

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		log.WarnMap("[UserDocumentHandler.GetDocumentsByUserID] 2. Missing auth user context", map[string]any{"path": c.Path()})
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	userID, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		log.WarnMap("[UserDocumentHandler.GetDocumentsByUserID] 2.1. Invalid user_id in path", map[string]any{"user_id_param": c.Params("user_id"), "path": c.Path()})
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}
	if userID != *rc.UserID && rc.Role != "ADMIN" {
		log.WarnMap("[UserDocumentHandler.GetDocumentsByUserID] 2.2. Forbidden access to another user's documents", map[string]any{"request_user_id": userID.String(), "auth_user_id": rc.UserID.String(), "role": rc.Role})
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	log.InfoMap("[UserDocumentHandler.GetDocumentsByUserID] 3. Fetching documents from service", map[string]any{"user_id": userID.String()})
	docs, err := h.service.GetDocumentsByUserID(ctx, userID)
	if err != nil {
		log.ErrorMap("[UserDocumentHandler.GetDocumentsByUserID] 4. Service error while fetching documents", map[string]any{"user_id": userID.String(), "err": err.Error()})
		return apperr.SendAppError(c, err)
	}

	log.InfoMap("[UserDocumentHandler.GetDocumentsByUserID] 5. Returning documents", map[string]any{"user_id": userID.String(), "count": len(docs)})
	return c.Status(fiber.StatusOK).JSON(docs)
}

func (h *UserDocumentHandler) GetDocumentsMe(c *fiber.Ctx) error {
	log := shared.GetLogger()
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	log.Info("[UserDocumentHandler.GetDocumentsMe] 1. Processing request for /documents/me")

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	sinceVersion, err := strconv.ParseUint(c.Query("since_version", "0"), 10, 64)
	if err != nil {
		sinceVersion = 0
	}
	ifNoneMatch := c.Get(fiber.HeaderIfNoneMatch)

	log.InfoMap("[UserDocumentHandler.GetDocumentsMe] 3. Fetching documents for current user", map[string]any{"user_id": rc.UserID.String(), "role": rc.Role, "since_version": sinceVersion, "if_none_match": ifNoneMatch})
	docs, etag, stateVersion, notModified, err := h.service.GetDocumentsByUserIDWithSyncState(ctx, *rc.UserID, sinceVersion, ifNoneMatch)
	if err != nil {
		log.ErrorMap("[UserDocumentHandler.GetDocumentsMe] 4. Service error while fetching current user documents", map[string]any{"user_id": rc.UserID.String(), "err": err.Error()})
		return apperr.SendAppError(c, err)
	}
	if etag != "" {
		c.Set(fiber.HeaderETag, strconv.Quote(etag))
	}
	c.Set("X-Document-State-Version", strconv.FormatUint(stateVersion, 10))
	if notModified {
		log.InfoMap("[UserDocumentHandler.GetDocumentsMe] 5. Returning 304 Not Modified", map[string]any{"user_id": rc.UserID.String(), "state_version": stateVersion, "etag": etag})
		return c.SendStatus(fiber.StatusNotModified)
	}

	log.InfoMap("[UserDocumentHandler.GetDocumentsMe] 6. Returning current user documents", map[string]any{"user_id": rc.UserID.String(), "count": len(docs), "state_version": stateVersion, "etag": etag})
	return c.Status(fiber.StatusOK).JSON(docs)
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
