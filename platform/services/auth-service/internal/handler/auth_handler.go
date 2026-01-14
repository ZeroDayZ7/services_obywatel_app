package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/constants"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/auth-service/internal/http"
	service "github.com/zerodayz7/platform/services/auth-service/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
	cache       *redis.Cache
	cfg         *viper.Config
}

func NewAuthHandler(authService service.AuthService, cache *redis.Cache, cfg *viper.Config) *AuthHandler { // USUNIĘTO *
	return &AuthHandler{
		authService: authService,
		cache:       cache,
		cfg:         cfg,
	}
}

// #region LOGIN
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()
	log := shared.GetLogger()

	body := c.Locals("validatedBody").(schemas.LoginRequest)
	rc := reqctx.MustFromFiber(c)

	// 2. Pobierz DeviceID (fingerprint)s
	fingerprint := rc.DeviceID

	if fingerprint == "" {
		return apperr.SendAppError(c, apperr.ErrInvalidDeviceFingerprint)
	}

	response, err := h.authService.AttemptLogin(ctx, body.Email, []byte(body.Password), fingerprint)
	if err != nil {
		log.WarnObj("Login failed", map[string]any{"email": body.Email, "err": err.Error()})
		return apperr.SendAppError(c, err)
	}

	log.InfoMap("Login successful", map[string]any{"email": body.Email})
	return c.JSON(response)
}

// #region VerifyDevice
func (h *AuthHandler) VerifyDevice(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	// 1. Dane z Body (od Użytkownika)
	var body schemas.VerifyDeviceRequest
	if err := c.BodyParser(&body); err != nil {
		return apperr.SendAppError(c, apperr.ErrInvalidRequestBody)
	}

	// 2. Dane z Contextu (Zaufane, od Gatewaya przez middleware)
	rc := reqctx.MustFromFiber(c)

	// 3. Delegacja - przekazujemy czyste parametry
	// Wyciągamy DeviceID (fingerprint), który u Ciebie jest w RequestContext
	response, err := h.authService.VerifyDeviceSignature(
		ctx,
		rc.UserID.String(),
		rc.SessionID,
		body.Signature,
		rc.DeviceID,
	)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.JSON(response)
}

// #region REGISTER DEVICE
func (h *AuthHandler) RegisterDevice(c *fiber.Ctx) error {
	log := shared.GetLogger()
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	body := c.Locals("validatedBody").(schemas.RegisterDeviceRequest)
	response, err := h.authService.RegisterDevice(
		ctx,
		*rc.UserID,
		rc.SessionID,
		rc.IP,
		body,
	)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	log.DebugJSON("response", response)

	return c.Status(fiber.StatusOK).JSON(response)
}

// #region VERIFY 2 FA
func (h *AuthHandler) Verify2FA(c *fiber.Ctx) error {
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(schemas.TwoFARequest)
	fingerprint := c.Get(constants.HeaderDeviceFingerprint)

	// Zerowanie kodu z pamięci (Security)
	defer func() {
		if len(body.Code) > 0 {
			for i := range body.Code {
				body.Code[i] = 0
			}
			log.Debug("Sensitive 2FA code bytes cleared from RAM")
		}
	}()

	// Wywołanie logiki biznesowej
	response, err := h.authService.Verify2FA(
		c.Context(),
		body.Token,
		body.Code,
		fingerprint,
		c.IP(),
	)
	if err != nil {
		log.WarnObj("2FA failed", map[string]any{"token": body.Token, "err": err.Error()})
		return apperr.SendAppError(c, err)
	}

	return c.JSON(response)
}

// #region REFRESH TOKEN
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	// Używamy bezpiecznego kontekstu z timeoutem
	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	body := c.Locals("validatedBody").(schemas.RefreshTokenRequest)
	fingerprint := c.Get(constants.HeaderDeviceFingerprint)

	if fingerprint == "" {
		return apperr.SendAppError(c, apperr.ErrInvalidToken)
	}

	// Wywołanie logiki biznesowej
	response, err := h.authService.RefreshToken(ctx, body.RefreshToken, fingerprint)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	return c.JSON(response)
}

// #region LOGOUT
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	log := shared.GetLogger()

	// Pobranie zwalidowanego kontekstu (skoro register tak robi, my też)
	rc := reqctx.MustFromFiber(c)
	if rc.UserID == nil {
		return apperr.SendAppError(c, apperr.ErrUnauthorized)
	}

	// Wyciągamy sessionID z headerów (bo to specyficzny identyfikator tej konkretnej sesji)
	sessionID := c.Get("X-Session-Id")
	if sessionID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing session ID")
	}

	// Wywołanie serwisu z "czystymi" danymi z rc (Request Context)
	// rc.UserID jest już typu *uuid.UUID, więc robimy dereferencję *rc.UserID
	err := h.authService.Logout(c.Context(), *rc.UserID, sessionID, rc.DeviceID)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	log.InfoMap("Logout successful", map[string]any{
		"user_id":    rc.UserID,
		"session_id": sessionID,
	})

	return c.Status(fiber.StatusOK).JSON(http.LogoutResponse{
		Message: "Logged out successfully",
	})
}

// #region REGISTER
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(schemas.RegisterRequest)

	// Próba rejestracji użytkownika
	user, err := h.authService.Register(body.Username, body.Email, body.Password)
	if err != nil {
		if appErr, ok := err.(*apperr.AppError); ok {
			apperr.AttachRequestMeta(c, appErr, "requestID")
			return appErr
		}
		return apperr.ErrInternal
	}

	// Przygotowanie minimalistycznej odpowiedzi DTO
	response := http.RegisterResponse{
		Success: true,
	}

	// Logujemy fakt rejestracji (zachowując ID w logach serwera dla audytu)
	log.InfoMap("User account created successfully", map[string]any{
		"email":   user.Email,
		"user_id": user.ID,
	})

	return c.Status(fiber.StatusCreated).JSON(response)
}
