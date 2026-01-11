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
// 1️⃣ Wyślij kod resetu
func (h *ResetHandler) SendResetCode(c *fiber.Ctx) error {
	body := c.Locals("validatedBody").(schemas.ResetPasswordRequest)

	token, err := h.resetService.StartResetProcess(c.Context(), body.Value)
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
	// 1. Pobranie zwalidowanych danych z Locals (zakładając użycie Twojego middleware)
	// Jeśli nie używasz middleware, użyj: req := new(schemas.FinalizeResetRequest); c.BodyParser(req)
	req, ok := c.Locals("validatedBody").(*schemas.FinalizeResetRequest)
	if !ok {
		// Fallback jeśli middleware nie dostarczył danych
		req = new(schemas.FinalizeResetRequest)
		if err := c.BodyParser(req); err != nil {
			return errors.SendAppError(c, errors.ErrInvalidRequest)
		}
	}

	// 2. Mapowanie danych urządzenia na DTO serwisu
	device := service.DeviceInfo{
		Fingerprint: req.Fingerprint,
		PublicKey:   req.PublicKey,
		DeviceName:  req.DeviceName,
		Platform:    req.Platform,
		IP:          c.IP(), // Pobieramy IP bezpośrednio z kontekstu Fiber
	}

	// 3. Wywołanie logiki biznesowej w serwisie
	// Przekazujemy context, token, nowe hasło, podpis oraz dane urządzenia
	err := h.resetService.FinalizeReset(
		c.Context(),
		req.Token,
		req.Password,
		req.Signature,
		device,
	)
	// 4. Obsługa błędów z serwisu
	if err != nil {
		return errors.SendAppError(c, err)
	}

	// 5. Sukces
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Password has been reset successfully",
	})
}
