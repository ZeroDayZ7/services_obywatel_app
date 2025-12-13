package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
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
	var doc model.UserDocument
	if err := c.BodyParser(&doc); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.service.CreateDocument(&doc); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(doc)
}

// GET /documents/me
func (h *UserDocumentHandler) GetDocumentsMe(c *fiber.Ctx) error {
	userIDHeader := c.Get("X-User-ID")
	if userIDHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing X-User-ID"})
	}

	userID, err := strconv.ParseUint(userIDHeader, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user ID"})
	}

	docs, err := h.service.GetDocumentsByUserID(uint(userID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch documents"})
	}

	return c.JSON(docs)
}
