package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/events"
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
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(validator.ResetPasswordRequest)

	// Pobranie użytkownika po emailu z pola Value
	user, err := h.authService.GetUserByEmail(ctx, body.Value)
	if err != nil {
		log.WarnObj("User not found", body.Value)
		// użycie naszego AppError zamiast ręcznego JSON-a
		return errors.SendAppError(c, errors.ErrEmailIsSendIfExists)
	}

	token := shared.GenerateUuidV7()
	code := fmt.Sprintf("%06d", shared.RandInt(100000, 999999))
	hashed, _ := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)

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

	response := validator.ResetSendResponse{
		Success:    true,
		ResetToken: token,
	}

	log.Debug("Reset code sent", map[string]any{
		"code":        code,
		"email":       user.Email,
		"reset_token": token,
	})

	return c.JSON(response)
}

// #region VERIFY RESET CODE
// 2️⃣ Weryfikacja kodu
func (h *ResetHandler) VerifyResetCode(c *fiber.Ctx) error {
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(validator.ResetCodeVerifyRequest)

	// DEBUG: Sprawdzenie co przyszło w żądaniu
	log.DebugMap("Processing VerifyResetCode", map[string]any{
		"token_from_body": body.Token,
		"code_length":     len(body.Code),
	})

	key := fmt.Sprintf("reset:password:%s", body.Token)

	// DEBUG: Sprawdzenie klucza w Redis
	log.DebugObj("Fetching from cache with key", key)
	data, err := h.cache.Get(c.Context(), key)
	if err != nil {
		log.WarnMap("Reset session not found", map[string]any{
			"token": body.Token,
			"error": err.Error(),
		})
		return errors.SendAppError(c, errors.ErrResetSessionNotFound)
	}

	var session ResetSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		log.ErrorObj("Failed to unmarshal session data", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// DEBUG: Sprawdzenie stanu sesji przed weryfikacją
	log.DebugMap("Session data retrieved", map[string]any{
		"user_id":  session.UserID,
		"attempts": session.Attempts,
	})

	if session.Attempts >= 5 {
		log.WarnObj("Max reset attempts reached for user", session.UserID)
		return errors.SendAppError(c, errors.Err2FALocked) // Używamy Twojego zdefiniowanego błędu blokady
	}

	// DEBUG: Porównywanie kodów (bcrypt)
	log.Debug("Comparing bcrypt hash with provided code")
	if err := bcrypt.CompareHashAndPassword([]byte(session.CodeHash), []byte(body.Code)); err != nil {
		session.Attempts++
		updated, _ := json.Marshal(session)
		h.cache.Set(c.Context(), key, updated, 5*time.Minute)

		log.WarnMap("Invalid code attempt", map[string]any{
			"user_id":      session.UserID,
			"new_attempts": session.Attempts,
		})
		return errors.SendAppError(c, errors.ErrInvalidResetCode)
	}

	// Sukces - generujemy challenge
	session.Challenge = shared.GenerateUuidV7()
	session.Verified = true
	updated, _ := json.Marshal(session)

	log.DebugObj("Code verified, saving session with challenge", session.Challenge)
	h.cache.Set(c.Context(), key, updated, 5*time.Minute)

	response := validator.ResetVerifyResponse{
		Success:    true,
		ResetToken: session.Token,
		UserID:     session.UserID,
		Challenge:  session.Challenge,
	}

	log.InfoMap("Reset code verified successfully", map[string]any{
		"user_id": session.UserID,
		"token":   session.Token,
	})

	return c.JSON(response)
}

// #region FINAL RESET PASSWORD
// 3️⃣ Finalny reset hasła

func (h *ResetHandler) ResetPassword(c *fiber.Ctx) error {
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(validator.ResetPasswordFinalRequest)

	// 1. Pobranie sesji resetu z cache
	key := fmt.Sprintf("reset:password:%s", body.Token)
	data, err := h.cache.Get(c.Context(), key)
	if err != nil {
		return errors.SendAppError(c, errors.ErrResetSessionNotFound)
	}

	var session ResetSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// Sprawdzenie czy kod został wcześniej zweryfikowany przez /verify
	if !session.Verified {
		return errors.SendAppError(c, errors.ErrUnauthorized)
	}

	userUUID, err := uuid.Parse(session.UserID)
	if err != nil {
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 2. Obsługa urządzenia i klucza publicznego
	var publicKeyToVerify string
	ctx := c.Context()
	device, err := h.authService.GetDeviceByFingerprint(ctx, userUUID, body.Fingerprint)

	if err != nil {
		// Scenariusz: Nowa instalacja aplikacji / brak urządzenia w bazie
		log.InfoMap("New device detected during password reset", map[string]any{
			"user_id":     userUUID,
			"fingerprint": body.Fingerprint,
		})

		if body.PublicKey == "" {
			return errors.SendAppError(c, errors.ErrUntrustedDevice)
		}

		deviceName := body.DeviceName
		if deviceName == "" {
			deviceName = "Unknown Device (Reset)"
		}

		platformName := body.Platform
		if platformName == "" {
			platformName = "unknown"
		}

		err = h.authService.RegisterUserDevice(
			c.Context(),
			userUUID,
			body.Fingerprint,
			body.PublicKey,
			deviceName,
			platformName,
			false,
			c.IP(),
		)
		if err != nil {
			log.ErrorObj("Failed to register temporary device during reset", err)
			return errors.SendAppError(c, errors.ErrInternal)
		}
		publicKeyToVerify = body.PublicKey
	} else {
		// Scenariusz: Znane urządzenie
		publicKeyToVerify = device.PublicKey
	}

	// 3. Weryfikacja podpisu kryptograficznego
	// Challenge zbudowany z unikalnego wyzwania sesji i kodu OTP
	challenge := fmt.Sprintf("%s|%s", session.Challenge, body.Code)

	isValidSignature := shared.VerifyEd25519Signature(
		publicKeyToVerify,
		challenge,
		body.Signature,
	)

	if !isValidSignature {
		log.ErrorMap("Invalid password reset signature", map[string]any{
			"user_id": userUUID,
			"fpt":     body.Fingerprint,
		})
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid security signature",
		})
	}

	// 4. Finalizacja: Zmiana hasła w bazie danych
	if err := h.authService.UpdatePassword(
		c.Context(),
		userUUID,
		body.NewPassword,
	); err != nil {
		log.ErrorObj("Failed to update password", err)
		return errors.SendAppError(c, errors.ErrInternal)
	}

	// 5. Sprzątanie i Eventy
	h.cache.Del(c.Context(), key)

	emitter := events.NewEmitter(h.cache, "auth-service")
	emitter.Emit(
		c.Context(),
		events.PasswordChanged,
		userUUID.String(),
		events.WithIP(c.IP()),
		events.WithMetadata(map[string]any{
			"fingerprint": body.Fingerprint,
			"method":      "reset_via_device_signature",
			"new_device":  err != nil, // true jeśli urządzenie było rejestrowane teraz
		}),
		events.WithFlags(events.EventFlags{
			Audit:  true,
			Notify: true,
		}),
	)

	log.InfoMap("Password reset finalized successfully", map[string]any{
		"user_id": userUUID,
	})

	return c.JSON(validator.ResetFinalResponse{
		Success: true,
	})
}
