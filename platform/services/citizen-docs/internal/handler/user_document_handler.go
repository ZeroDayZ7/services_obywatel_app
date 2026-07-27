package handler

import (
	"context"
	"encoding/json"
	"io"
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

// #region CREATE DOCUMENT
func (h *UserDocumentHandler) CreateDocument(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()
	log := shared.GetLogger()

	// 1. Pobieramy metadane z formularza
	metaStr := c.FormValue("meta")
	var meta model.DocumentMeta
	if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	// 2. Parsujemy profile_id jako UUID
	profileID, err := uuid.Parse(c.FormValue("profile_id"))
	if err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}
	docType := model.DocumentType(c.FormValue("type"))

	// 3. Pobieramy i czytamy pliki front/back
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

	// 4. Wywołanie logiki biznesowej
	err = h.service.CreateDocument(
		ctx,
		&meta,
		frontBytes,
		backBytes,
		profileID,
		docType,
	)
	if err != nil {
		log.ErrorObj("Failed to create user document", map[string]any{"profile_id": profileID, "err": err.Error()})
		return apperr.SendAppError(c, err)
	}

	log.InfoMap("Document created successfully", map[string]any{"profile_id": profileID, "type": docType})
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

	// Tworzymy dokładną odpowiedź API
	response := map[string]any{
		"user_id": rc.UserID.String(),
		"count":   len(docs),
		"docs":    docs,
	}

	// DEBUG - pokazuje dokładny JSON
	debugJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.ErrorObj("Failed to marshal documents response", map[string]any{
			"error": err.Error(),
		})
	} else {
		log.DebugInfo("GET /documents/me JSON RESPONSE", map[string]any{
			"json": string(debugJSON),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// #region HELPERS
func readFileFromForm(c *fiber.Ctx, fieldName string) ([]byte, error) {
	fileHeader, err := c.FormFile(fieldName)
	if err != nil {
		return nil, nil // Plik jest opcjonalny
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return data, nil
}
