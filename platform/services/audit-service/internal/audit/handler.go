package audit

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

	limitStr := c.Query("limit", "50")
	offsetStr := c.Query("offset", "0")
	h.logger.DebugMap("Query params", map[string]any{"limit": limitStr, "offset": offsetStr})

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		h.logger.WarnMap("Invalid limit query param, using default 50", map[string]any{"input": limitStr})
		limit = 50
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		h.logger.WarnMap("Invalid offset query param, using default 0", map[string]any{"input": offsetStr})
		offset = 0
	}

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
	h.logger.DebugMap("Received request for user logs", map[string]any{"user_id_param": idParam})

	uID, err := uuid.Parse(idParam)
	if err != nil {
		h.logger.WarnMap("Invalid UUID format in request", map[string]any{"input": idParam})
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user uuid format"})
	}
	h.logger.InfoMap("Parsed user UUID successfully", map[string]any{"user_id": uID})

	logs, err := h.svc.GetLogsByUserID(c.Context(), uID)
	if err != nil {
		h.logger.ErrorObj("Failed to fetch user logs", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch user logs"})
	}

	h.logger.InfoMap("Returned user logs successfully", map[string]any{"user_id": uID, "count": len(logs)})
	return c.JSON(logs)
}

// GetLogDetails zwraca jeden konkretny log
// GET /audit/:id
func (h *AuditHandler) GetLogDetails(c *fiber.Ctx) error {
	idParam := c.Params("id")
	h.logger.InfoMap("Fetching log details", map[string]any{"log_id_param": idParam})

	logID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		h.logger.WarnMap("Invalid log ID format", map[string]any{"input": idParam})
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	h.logger.DebugMap("Parsed log ID successfully", map[string]any{"log_id": logID})

	logEntry, err := h.svc.GetLogByID(c.Context(), logID)
	if err != nil {
		h.logger.WarnMap("Log not found", map[string]any{"log_id": logID, "error": err})
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	h.logger.InfoMap("Returned log details successfully", map[string]any{"log_id": logID})
	return c.JSON(logEntry)
}

// ListLogsByAction filtruje logi po typie akcji
// GET /audit/action/:action
func (h *AuditHandler) ListLogsByAction(c *fiber.Ctx) error {
	action := c.Params("action")
	h.logger.DebugMap("Received request for logs by action", map[string]any{"action_param": action})

	if action == "" {
		h.logger.Warn("Action param is empty")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "action is required"})
	}

	logs, err := h.svc.GetLogsByAction(c.Context(), action)
	if err != nil {
		h.logger.ErrorObj("Failed to fetch logs by action", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch logs by action"})
	}

	h.logger.InfoMap("Returned logs by action successfully", map[string]any{"action": action, "count": len(logs)})
	return c.JSON(logs)
}
