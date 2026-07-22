package middleware

import (
	"github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	appcontext "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

func ContextBuilder(container *di.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()

		// 1. Pobieramy sparowany token JWT wrzucony wcześniej przez middleware jwtware
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok || token == nil {
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		sid, _ := claims["sid"].(string)
		if sid == "" {
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		// 2. Pobieramy i automatycznie unmarshalujemy sesję z Redisa
		sessionData, err := container.Cache.GetSession(c.Context(), sid)
		if err != nil {
			log.Warn("Session missing or expired in Redis", "sid", sid, "err", err)
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		parsedUserID, err := uuid.Parse(sessionData.UserID)
		if err != nil {
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		// 3. Budowa uniwersalnego kontekstu żądania
		reqCtx := &appcontext.RequestContext{
			UserID:      &parsedUserID,
			SessionID:   sid,
			Role:        sessionData.Role,
			Permissions: sessionData.Permissions,
		}

		c.Locals("requestContext", reqCtx)
		return c.Next()
	}
}
