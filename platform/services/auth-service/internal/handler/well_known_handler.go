package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	service "github.com/zerodayz7/platform/services/auth-service/internal/service"
)

type WellKnownHandler struct {
	keyService service.KeyService
}

func NewWellKnownHandler(keyService service.KeyService) *WellKnownHandler {
	return &WellKnownHandler{
		keyService: keyService,
	}
}

func (h *WellKnownHandler) GetJWKS(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer cancel()

	jwks, err := h.keyService.GetJWKS(ctx)
	if err != nil {
		return apperr.SendAppError(c, err)
	}

	// Wskazówka: Warto ustawić nagłówki Cache-Control dla kluczy publicznych
	c.Set("Cache-Control", "public, max-age=3600")
	return c.JSON(jwks)
}