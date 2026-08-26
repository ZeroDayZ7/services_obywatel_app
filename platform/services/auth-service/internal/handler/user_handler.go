package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/constants"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/utils"
	"github.com/zerodayz7/platform/services/auth-service/internal/http"
	"github.com/zerodayz7/platform/services/auth-service/internal/service"
)

type UserHandler struct {
	userService service.UserService
}

//#region NewUserHandler
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// region GetSessions
//#region GetSessions
func (h *UserHandler) GetSessions(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	// 1. Dane z HTTP
	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}
	fingerprint := c.Get(constants.HeaderDeviceFingerprint)

	// 2. Wywołanie serwisu
	sessions, err := h.userService.GetSessions(ctx, userID, fingerprint)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 3. Odpowiedź
	return c.JSON(sessions)
}

// region TerminateSession
// POST /user/sessions/terminate
//#region TerminateSession
func (h *UserHandler) TerminateSession(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	type request struct {
		SessionID uuid.UUID `json:"session_id"` // <-- zmiana z uint na uuid.UUID
	}
	var req request
	if err := c.BodyParser(&req); err != nil {
		return errors.SendAppError(c, errors.ErrInvalidRequest)
	}

	if err := h.userService.RevokeSession(ctx, userID, req.SessionID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	return c.JSON(http.TerminateSessionResponse{Status: "success"})
}
