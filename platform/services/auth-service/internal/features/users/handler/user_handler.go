package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	authRepo "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/repository"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/users/service"
)

type UserHandler struct {
	userService *service.UserService
	authRepo    authRepo.RefreshTokenRepository
}

func NewUserHandler(
	userService *service.UserService,
	authRepo authRepo.RefreshTokenRepository,
) *UserHandler {
	return &UserHandler{
		userService: userService,
		authRepo:    authRepo,
	}
}

// GET /user/sessions
func (h *UserHandler) GetSessions(c *fiber.Ctx) error {
	log := shared.GetLogger()

	userIDStr := c.Get("X-User-Id")
	if userIDStr == "" {
		log.Warn("GetSessions: missing X-User-Id header")
		return c.Status(fiber.StatusUnauthorized).
			JSON(fiber.Map{"error": "missing user id"})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.WarnObj("GetSessions: invalid user id format", userIDStr)
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{
				"error": "invalid user id format",
				"code":  "INVALID_UUID",
			})
	}

	log.DebugMap("GetSessions: fetching sessions", map[string]any{
		"user_id": userID,
	})

	sessions, err := h.authRepo.GetSessionsWithDeviceData(uuid.UUID(userID))
	if err != nil {
		log.ErrorObj("GetSessions: repository error", err)
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "could not fetch sessions"})
	}

	log.InfoMap("GetSessions: sessions fetched", map[string]any{
		"user_id":        userID,
		"sessions_count": len(sessions),
	})

	return c.JSON(sessions)
}

// POST /user/sessions/terminate
func (h *UserHandler) TerminateSession(c *fiber.Ctx) error {
	log := shared.GetLogger()

	userIDStr := c.Get("X-User-Id")
	if userIDStr == "" {
		log.Warn("TerminateSession: missing X-User-Id header")
		return c.Status(fiber.StatusUnauthorized).
			JSON(fiber.Map{"error": "missing user id"})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.WarnObj("GetSessions: invalid user id format", userIDStr)
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{
				"error": "invalid user id format",
				"code":  "INVALID_UUID",
			})
	}

	type request struct {
		SessionID uint `json:"session_id"`
	}

	var req request
	if err := c.BodyParser(&req); err != nil {
		log.WarnObj("TerminateSession: invalid body", err)
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "invalid request"})
	}

	log.InfoMap("TerminateSession: terminating session", map[string]any{
		"user_id":    userID,
		"session_id": req.SessionID,
	})

	err = h.authRepo.DeleteByID(req.SessionID, userID)
	if err != nil {
		log.ErrorMap("TerminateSession: failed to revoke session", map[string]any{
			"user_id":    userID,
			"session_id": req.SessionID,
			"error":      err,
		})
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "could not terminate session"})
	}

	log.InfoMap("TerminateSession: session terminated", map[string]any{
		"user_id":    userID,
		"session_id": req.SessionID,
	})

	return c.JSON(fiber.Map{"status": "success"})
}
