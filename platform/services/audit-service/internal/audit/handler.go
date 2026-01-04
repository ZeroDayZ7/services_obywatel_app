package audit

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
)

type AuditHandler struct {
	svc    *AuditService
	logger *shared.Logger
}

func NewAuditHandler(svc *AuditService, l *shared.Logger) *AuditHandler {
	return &AuditHandler{
		svc:    svc,
		logger: l,
	}
}

// ListLogs zwraca listę wszystkich logów (dla Admina)
func (h *AuditHandler) ListLogs(c *fiber.Ctx) error {
	h.logger.Info("Admin requested full audit logs list")

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	logs, err := h.svc.GetAllLogs(c.Context(), int32(limit), int32(offset))
	if err != nil {
		h.logger.ErrorObj("Failed to fetch logs from DB", err)
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}

	h.logger.InfoMap("Successfully returned logs", map[string]any{"count": len(logs)})
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
	h.logger.InfoMap("Fetching log details", map[string]any{
		"log_id": idParam,
	})

	logID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	logEntry, err := h.svc.GetLogByID(c.Context(), logID)
	if err != nil {
		h.logger.WarnMap("Log not found", map[string]any{"log_id": logID})
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
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
