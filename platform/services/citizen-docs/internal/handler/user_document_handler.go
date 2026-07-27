// platform/services/citizen-docs/internal/handler/user_document_handler.go

package handler

import (
	"encoding/json"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/service"
)

type UserDocumentHandler struct {
	service *service.UserDocumentService
}

func NewUserDocumentHandler(s *service.UserDocumentService) *UserDocumentHandler {
	return &UserDocumentHandler{service: s}
}

// POST /documents
func (h *UserDocumentHandler) CreateDocument(c *fiber.Ctx) error {
	// 1. Pobieramy metadane
	metaStr := c.FormValue("meta")
	var meta model.DocumentMeta
	if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid meta format"})
	}

	// 2. Parsujemy profile_id jako UUID
	profileID, err := uuid.Parse(c.FormValue("profile_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid profile_id UUID format"})
	}
	docType := model.DocumentType(c.FormValue("type"))

	// 3. Pobieramy i czytamy pliki
	frontBytes, err := readFileFromForm(c, "front")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "failed to read front file: " + err.Error()})
	}

	backBytes, err := readFileFromForm(c, "back")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "failed to read back file: " + err.Error()})
	}

	// 4. Wywołanie serwisu
	err = h.service.CreateDocument(
		c.Context(),
		&meta,
		frontBytes,
		backBytes,
		profileID,
		docType,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "created"})
}

// GET /documents/me
func (h *UserDocumentHandler) GetDocumentsMe(c *fiber.Ctx) error {
	profileIDStr := c.Get("X-Profile-ID")
	if profileIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing X-Profile-ID"})
	}

	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid profile ID UUID format"})
	}

	docs, err := h.service.GetDocumentsByProfileID(c.Context(), profileID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch documents"})
	}

	return c.JSON(docs)
}

// Pomocnicza funkcja do bezpiecznego czytania plików z formularza
func readFileFromForm(c *fiber.Ctx, fieldName string) ([]byte, error) {
	fileHeader, err := c.FormFile(fieldName)
	if err != nil {
		return nil, nil // Plik opcjonalny, brak błędu jeśli go nie ma
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
