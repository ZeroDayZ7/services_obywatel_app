package audit

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type AuditHandler struct {
	svc *AuditService
}

func NewAuditHandler(svc *AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// ListLogs zwraca listę wszystkich logów (dla Admina)
// GET /audit
func (h *AuditHandler) ListLogs(c *fiber.Ctx) error {
	// Opcjonalnie: pobieranie limitu i offsetu z query params
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	logs, err := h.svc.GetAllLogs(c.Context(), int32(limit), int32(offset))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch logs",
		})
	}

	return c.JSON(logs)
}

// ListUserLogs zwraca logi konkretnego użytkownika
// GET /audit/user/:id
func (h *AuditHandler) ListUserLogs(c *fiber.Ctx) error {
	idParam := c.Params("id")
	userID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	logs, err := h.svc.GetLogsByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch user logs"})
	}

	return c.JSON(logs)
}

// GetLogDetails zwraca jeden konkretny log
// GET /audit/:id
func (h *AuditHandler) GetLogDetails(c *fiber.Ctx) error {
	idParam := c.Params("id")
	logID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid log id"})
	}

	logEntry, err := h.svc.GetLogByID(c.Context(), logID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "log not found"})
	}

	return c.JSON(logEntry)
}

// ListLogsByAction filtruje logi po typie akcji
// GET /audit/action/:action
func (h *AuditHandler) ListLogsByAction(c *fiber.Ctx) error {
	action := c.Params("action")
	if action == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "action is required"})
	}

	logs, err := h.svc.GetLogsByAction(c.Context(), action)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch logs by action"})
	}

	return c.JSON(logs)
}
