package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/constants"
	cts "github.com/zerodayz7/platform/pkg/context"
	"github.com/zerodayz7/platform/pkg/errors"
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
	fingerprint := c.Get(constants.HeaderDeviceFingerprint)

	defer func() {
		if len(body.Password) > 0 {
			for i := range body.Password {
				body.Password[i] = 0
			}
		}
	}()

	response, err := h.authService.AttemptLogin(ctx, body.Email, []byte(body.Password), fingerprint)
	if err != nil {
		log.WarnObj("Login failed", map[string]any{"email": body.Email, "err": err.Error()})
		return errors.SendAppError(c, err)
	}

	log.InfoMap("Login successful", map[string]any{"email": body.Email})
	return c.JSON(response)
}

// #region REGISTER DEVICE
func (h *AuthHandler) RegisterDevice(c *fiber.Ctx) error {
	log := shared.GetLogger()
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	rc := cts.MustFromFiber(c)
	if rc.UserID == nil {
		return errors.SendAppError(c, errors.ErrUnauthorized)
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
		return errors.SendAppError(c, err)
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
		return errors.SendAppError(c, err)
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
		return errors.SendAppError(c, errors.ErrInvalidToken)
	}

	// Wywołanie logiki biznesowej
	response, err := h.authService.RefreshToken(ctx, body.RefreshToken, fingerprint)
	if err != nil {
		return errors.SendAppError(c, err)
	}

	return c.JSON(response)
}

// #region LOGOUT
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	log := shared.GetLogger()
	body := c.Locals("validatedBody").(schemas.RefreshTokenRequest)

	// Pobranie danych z headerów (dodanych przez Middleware lub klienta)
	userID := c.Get("X-User-Id")
	sessionID := c.Get("X-Session-Id")

	if userID == "" || sessionID == "" || body.RefreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing required session data")
	}

	// Wywołanie serwisu
	err := h.authService.Logout(c.Context(), body.RefreshToken, userID, sessionID)
	if err != nil {
		return errors.SendAppError(c, err)
	}

	log.InfoMap("Logout successful", map[string]any{
		"user_id":    userID,
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
		if appErr, ok := err.(*errors.AppError); ok {
			errors.AttachRequestMeta(c, appErr, "requestID")
			return appErr
		}
		return errors.ErrInternal
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
