package middleware

import (
	"fmt"
	"slices"

	"github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/constants"
	"github.com/zerodayz7/platform/pkg/context"
)

func ContextBuilder() fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// 1. Podstawowe dane dostępne dla KAŻDEGO requestu (nawet publicznego)
		ctx := &context.RequestContext{
			// Używamy Locals("requestid"), bo fiber middleware 'requestid' tam go wrzuca
			RequestID: fmt.Sprintf("%v", c.Locals("requestid")),
			IP:        c.IP(),
			DeviceID:  c.Get(constants.HeaderDeviceFingerprint),
		}

		// 2. Jeśli ścieżka jest publiczna, pomijamy wyciąganie danych usera
		if slices.Contains(constants.PublicPaths, path) {
			c.Locals("requestContext", ctx)
			return c.Next()
		}

		// 3. Dane usera (tylko dla chronionych tras)
		userToken, ok := c.Locals("user").(*jwt.Token)
		if ok && userToken != nil {
			claims, ok := userToken.Claims.(jwt.MapClaims)
			if ok {
				// Bezpieczne parsowanie UUID
				if uidStr, ok := claims["uid"].(string); ok {
					if uid, err := uuid.Parse(uidStr); err == nil {
						ctx.UserID = &uid
					}
				}

				// Pobieranie Session ID
				if sid, ok := claims["sid"].(string); ok {
					ctx.SessionID = sid
				}

				// Pobieranie Ról
				if roles, ok := claims["roles"].([]any); ok {
					for _, r := range roles {
						if roleStr, ok := r.(string); ok {
							ctx.Roles = append(ctx.Roles, roleStr)
						}
					}
				}
			}
		}

		// 4. Zapisujemy gotowy obiekt w Locals
		c.Locals("requestContext", ctx)
		return c.Next()
	}
}
