package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/utils"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/users/http"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/users/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GET /user/sessions
func (h *UserHandler) GetSessions(c *fiber.Ctx) error {
	// 1. Context i Logger
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()
	log := shared.GetLogger()

	// 2. Autoryzacja (UserID z middleware)
	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	// 3. Wywołanie serwisu
	sessions, err := h.userService.GetSessions(ctx, userID)
	if err != nil {
		log.ErrorObj("GetSessions service error", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 4. Mapowanie na DTO
	currentFingerprint := c.Get("X-Device-Fingerprint")
	var response []http.SessionResponse

	for _, s := range sessions {
		response = append(response, http.SessionResponse{
			ID:        s.SessionID,
			Device:    s.DeviceNameEncrypted,
			Platform:  s.Platform,
			IsCurrent: s.Fingerprint == currentFingerprint,
			CreatedAt: s.CreatedAt,
			LastUsed:  s.LastUsedAt,
		})
	}

	return c.JSON(response)
}

// POST /user/sessions/terminate
func (h *UserHandler) TerminateSession(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	userID, err := utils.GetUserID(c)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	type request struct {
		SessionID uint `json:"session_id"`
	}
	var req request
	if err := c.BodyParser(&req); err != nil {
		return errors.SendAppError(c, errors.ErrInvalidRequest)
	}

	// Wywołanie serwisu (przekazujemy już oczyszczone typy uint i UUID)
	if err := h.userService.RevokeSession(ctx, userID, req.SessionID); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	return c.JSON(http.TerminateSessionResponse{Status: "success"})
}
