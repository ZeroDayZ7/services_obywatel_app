package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/service"
	"github.com/zerodayz7/platform/services/auth-service/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

type ResetHandler struct {
	authService *service.AuthService
	cache       *redis.Cache
}

func NewResetHandler(authService *service.AuthService, cache *redis.Cache) *ResetHandler {
	return &ResetHandler{
		authService: authService,
		cache:       cache,
	}
}

// ResetSession w Redis – wzór jak w 2FA
type ResetSession struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	CodeHash string `json:"code"`  // hashed kod
	Token    string `json:"token"` // UUID token
	Attempts int    `json:"attempts"`
}

// 1️⃣ Wyślij kod resetu
func (h *ResetHandler) SendResetCode(c *fiber.Ctx) error {
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(validator.ResetPasswordRequest)

	// Pobranie użytkownika po emailu z pola Value
	user, err := h.authService.GetUserByEmail(body.Value)
	if err != nil {
		log.WarnObj("User not found", body.Value)
		// użycie naszego AppError zamiast ręcznego JSON-a
		return errors.SendAppError(c, errors.ErrEmailIsSendIfExists)
	}

	code := fmt.Sprintf("%06d", shared.RandInt(100000, 999999))
	hashed, _ := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	token := shared.GenerateUuid()

	session := ResetSession{
		UserID:   fmt.Sprint(user.ID),
		Email:    user.Email,
		CodeHash: string(hashed),
		Token:    token,
		Attempts: 0,
	}

	data, _ := json.Marshal(session)
	key := fmt.Sprintf("reset:password:%s", token)
	if err := h.cache.Set(c.Context(), key, data, 5*time.Minute); err != nil {
		log.ErrorObj("Failed to save reset session", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// TODO: wyślij kod mailem/SMS
	// h.authService.SendResetCodeEmail(user.Email, code)

	return c.JSON(fiber.Map{"success": true, "reset_token": token})
}

// 2️⃣ Weryfikacja kodu
func (h *ResetHandler) VerifyResetCode(c *fiber.Ctx) error {
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(validator.ResetCodeVerifyRequest)

	key := fmt.Sprintf("reset:password:%s", body.Token)
	data, err := h.cache.Get(c.Context(), key)
	if err != nil {
		log.WarnObj("Reset session not found", body.Token)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid or expired token"})
	}

	var session ResetSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		log.ErrorObj("Failed to unmarshal reset session", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal error"})
	}

	if session.Attempts >= 5 {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Too many attempts"})
	}

	if bcrypt.CompareHashAndPassword([]byte(session.CodeHash), []byte(body.Code)) != nil {
		session.Attempts++
		updated, _ := json.Marshal(session)
		h.cache.Set(c.Context(), key, updated, 5*time.Minute)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid code"})
	}

	// Kod prawidłowy – pozostaw token w Redis do finalnego resetu lub usuń jeśli wolisz
	return c.JSON(fiber.Map{"success": true, "reset_token": session.Token, "user_id": session.UserID})
}

// 3️⃣ Finalny reset hasła
func (h *ResetHandler) ResetPassword(c *fiber.Ctx) error {
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(validator.ResetPasswordFinalRequest)

	key := fmt.Sprintf("reset:password:%s", body.Token)
	data, err := h.cache.Get(c.Context(), key)
	if err != nil {
		log.WarnObj("Reset session not found", body.Token)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid or expired token"})
	}

	var session ResetSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		log.ErrorObj("Failed to unmarshal reset session", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal error"})
	}

	uid, err := uuid.Parse(session.UserID)
	if err != nil {
		log.ErrorMap("Błędny format UUID w sesji", map[string]any{
			"userID": session.UserID,
			"error":  err.Error(),
		})
		return errors.SendAppError(c, errors.ErrInternal)
	}

	if err := h.authService.UpdatePassword(uuid.UUID(uid), body.NewPassword); err != nil {
		log.ErrorObj("Failed to update password", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to reset password"})
	}

	// Usuń sesję z Redis
	h.cache.Del(c.Context(), key)

	return c.JSON(fiber.Map{"success": true})
}
