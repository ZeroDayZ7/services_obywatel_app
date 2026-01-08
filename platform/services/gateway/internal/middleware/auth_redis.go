package middleware

import (
	"context"
	"encoding/json"
	"errors"

	"slices"

	"github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/zerodayz7/platform/pkg/shared"
)

// PublicPathsRedis - ścieżki, które NIE wymagają weryfikacji Redis
var PublicPathsRedis = []string{
	"/auth/login",
	"/auth/register",
	"/auth/refresh",
	"/auth/2fa-verify",
	"/auth/reset/verify",
	"/auth/reset/send",
	"/auth/reset/final",
	"/auth/2fa-resend",
	"/health",
}

type UserSession struct {
	UserID      string `json:"user_id"`
	Fingerprint string `json:"fingerprint"`
}

// AuthRedisMiddleware - middleware do weryfikacji sesji w Redis
func AuthRedisMiddleware(rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()
		path := c.Path()

		// 1. Skip public paths
		if slices.Contains(PublicPathsRedis, path) {
			return c.Next()
		}

		// 2. Pobierz fingerprint wysłany przez klienta (Dio Interceptor)
		clientFingerprint := c.Get("X-Device-Fingerprint")
		if clientFingerprint == "" {
			log.Warn("Missing X-Device-Fingerprint header")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "device identification missing"})
		}

		// 3. Wyciągnij sessionID z JWT (ustawionego wcześniej przez JWTMiddleware)
		jwtPayload := c.Locals("user")
		if jwtPayload == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		jwtToken := jwtPayload.(*jwt.Token)
		claims := jwtToken.Claims.(jwt.MapClaims)
		sessionID, _ := claims["sid"].(string)

		// 4. Pobierz dane sesji z Redis
		ctx := context.Background()
		jsonData, err := rdb.Get(ctx, "session:"+sessionID).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "session expired"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "redis error"})
		}

		// 5. PARSOWANIE JSON I WERYFIKACJA FINGERPRINTU
		var session UserSession
		if err := json.Unmarshal([]byte(jsonData), &session); err != nil {
			log.Error("Failed to unmarshal session from redis")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "session data corrupted"})
		}

		// --- KLUCZOWY MOMENT ---
		if session.Fingerprint != clientFingerprint {
			log.WarnMap("Fingerprint mismatch!", map[string]any{
				"sessionID": sessionID,
				"expected":  session.Fingerprint,
				"received":  clientFingerprint,
			})
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid device binding"})
		}

		// 6. Ustawienie danych dla downstreamu (Gateway przekaże to w nagłówkach do mikroserwisów)
		c.Locals("userID", session.UserID)
		c.Locals("sessionID", sessionID)

		// Opcjonalnie: ustaw nagłówki dla mikroserwisów, żeby wiedziały kto pyta
		c.Request().Header.Set("X-User-Id", session.UserID)
		c.Request().Header.Set("X-Session-Id", sessionID)

		return c.Next()
	}
}
