package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/services/auth-service/internal/http"
	"github.com/zerodayz7/platform/services/auth-service/internal/service"
)

type ResetHandler struct {
	resetService service.PasswordResetService
	cache        *redis.Cache
}

func NewResetHandler(resetService service.PasswordResetService, cache *redis.Cache) *ResetHandler {
	return &ResetHandler{
		resetService: resetService,
		cache:        cache,
	}
}

// #region STRUCT RESET SESSION
// ResetSession w Redis – wzór jak w 2FA
type ResetSession struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	CodeHash  string `json:"code"`
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
	Attempts  int    `json:"attempts"`
	Verified  bool   `json:"verified"`
}

// #region SEND RESET CODE
func (h *ResetHandler) SendResetCode(c *fiber.Ctx) error {
	body := c.Locals("validatedBody").(schemas.ResetPasswordRequest)

	token, err := h.resetService.StartResetProcess(
		c.Context(),
		body.AccountIdentifier,
		body.Value,
		body.Method,
	)
	if err != nil {
		return errors.SendAppError(c, err)
	}

	return c.JSON(http.ResetSendResponse{
		Success:    true,
		ResetToken: token,
	})
}

// #region VERIFY RESET CODE
func (h *ResetHandler) VerifyResetCode(c *fiber.Ctx) error {
	body := c.Locals("validatedBody").(schemas.ResetCodeVerifyRequest)

	session, err := h.resetService.VerifyCode(c.Context(), body.Token, body.Code)
	if err != nil {
		return errors.SendAppError(c, err)
	}

	return c.JSON(http.ResetVerifyResponse{
		Success:    true,
		ResetToken: session.Token,
		UserID:     session.UserID,
		Challenge:  session.Challenge,
	})
}

// #region FINAL RESET PASSWORD
func (h *ResetHandler) FinalizeReset(c *fiber.Ctx) error {
	req, ok := c.Locals("validatedBody").(*schemas.ResetPasswordFinalRequest)
	if !ok {
		req = new(schemas.ResetPasswordFinalRequest)
		if err := c.BodyParser(req); err != nil {
			return errors.SendAppError(c, errors.ErrInvalidRequest)
		}
	}

	err := h.resetService.FinalizeReset(
		c.UserContext(),
		req.Token,
		req.NewPassword,
	)
	if err != nil {
		return errors.SendAppError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Password has been reset successfully",
	})
}
