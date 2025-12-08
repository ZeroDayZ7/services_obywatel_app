package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/http-server/internal/model"
	"github.com/zerodayz7/http-server/internal/service"
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

// GET /documents/user/:id
func (h *UserDocumentHandler) GetDocumentsByUserID(c *fiber.Ctx) error {
	userIDParam := c.Params("id")
	userID, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	docs, err := h.service.GetDocumentsByUserID(uint(userID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(docs)
}
