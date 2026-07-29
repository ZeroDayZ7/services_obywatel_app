package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
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

// #region CREATE DOCUMENT
func (h *UserDocumentHandler) CreateDocument(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()
	log := shared.GetLogger()

	// 1. Pobieramy i walidujemy ID profilu oraz typ dokumentu
	profileID, err := uuid.Parse(c.FormValue("profile_id"))
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	typeCode := c.FormValue("type_code")
	if typeCode == "" {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	// 2. Parsujemy metadane z formularza
	metaStr := c.FormValue("meta")
	var meta model.DocumentMeta
	if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	// 3. Odczytujemy pola wymagane do weryfikacji offline (Signatures & Keys)
	sigBase64 := c.FormValue("issuer_signature")
	signingKeyID := c.FormValue("signing_key_id")
	revocationSerial := c.FormValue("revocation_serial")

	if sigBase64 == "" || signingKeyID == "" || revocationSerial == "" {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	issuerSignature, err := base64.StdEncoding.DecodeString(sigBase64)
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	// 4. Pobieramy pliki obrazów (front/back)
	frontBytes, err := readFileFromForm(c, "front")
	if err != nil {
		log.WarnObj("Failed to read front document file", map[string]any{"err": err.Error()})
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	backBytes, err := readFileFromForm(c, "back")
	if err != nil {
		log.WarnObj("Failed to read back document file", map[string]any{"err": err.Error()})
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	// 5. Wywołanie logiki biznesowej z nowym kontraktem
	err = h.service.CreateDocument(
		ctx,
		profileID,
		typeCode,
		&meta,
		frontBytes,
		backBytes,
		issuerSignature,
		signingKeyID,
		revocationSerial,
	)
	if err != nil {
		log.ErrorObj("Failed to create user document", map[string]any{"profile_id": profileID, "err": err.Error()})
		return apperr.SendAppError(c, err)
	}

	log.InfoMap("Document created successfully", map[string]any{"profile_id": profileID, "type_code": typeCode})
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "created"})
}

// #region GET DOCUMENTS ME
func (h *UserDocumentHandler) GetDocumentsMe(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	log := shared.GetLogger()
	rc := reqctx.MustFromFiber(c)

	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	docs, err := h.service.GetDocumentsByUserID(ctx, *rc.UserID)
	if err != nil {
		log.ErrorObj("Failed to fetch user documents", map[string]any{
			"user_id": rc.UserID,
			"error":   err.Error(),
		})
		return apperr.SendAppError(c, err)
	}

	response := mapper.ToUserDocumentResponses(docs)

	log.DebugInfo("GET /documents/me fetched successfully", map[string]any{
		"user_id": rc.UserID.String(),
		"count":   len(response),
	})

	return c.Status(fiber.StatusOK).JSON(response)
}

// #region GET DOCUMENTS SYNC (DELTA SYNC FOR MOBILE OFFLINE)
func (h *UserDocumentHandler) GetDocumentsSync(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	log := shared.GetLogger()
	rc := reqctx.MustFromFiber(c)

	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	sinceVersionStr := c.Query("since_version", "0")
	sinceVersion, err := strconv.ParseUint(sinceVersionStr, 10, 64)
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	docs, err := h.service.GetDocumentsSinceVersion(ctx, *rc.UserID, sinceVersion)
	if err != nil {
		log.ErrorObj("Failed to sync user documents", map[string]any{
			"user_id":       rc.UserID.String(),
			"since_version": sinceVersion,
			"error":         err.Error(),
		})
		return apperr.SendAppError(c, err)
	}

	response := mapper.ToUserDocumentResponses(docs)
	return c.Status(fiber.StatusOK).JSON(response)
}

// #endregion

// #region HELPERS
func readFileFromForm(c *fiber.Ctx, fieldName string) ([]byte, error) {
	fileHeader, err := c.FormFile(fieldName)
	if err != nil {
		return nil, nil // Plik opcjonalny
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}
