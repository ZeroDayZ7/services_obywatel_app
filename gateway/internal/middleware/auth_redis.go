package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"slices"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/zerodayz7/http-server/internal/shared/logger"
	"github.com/zerodayz7/http-server/internal/shared/security"
)

// PublicPathsRedis - ścieżki, które NIE wymagają autoryzacji JWT/Redis
var PublicPathsRedis = []string{
	"/auth/login",
	"/auth/register",
	"/auth/refresh",
	"/auth/2fa-verify",
	"/health",
}

// AuthRedisMiddleware - middleware do weryfikacji JWT i sesji w Redis
func AuthRedisMiddleware(rdb *redis.Client, accessSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := logger.GetLogger()
		path := c.Path()

		// 1. Sprawdź, czy ścieżka jest publiczna
		if slices.Contains(PublicPathsRedis, path) {
			log.DebugMap("Public path, skipping JWT/Redis check", map[string]string{
				"path": path,
			})
			return c.Next()
		}

		// 2. Pobierz token z nagłówka Authorization
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			log.WarnMap("Missing Authorization header", map[string]string{
				"path": path,
			})
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.WarnMap("Invalid Authorization header format", map[string]string{
				"header": authHeader,
				"path":   path,
			})
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization header"})
		}
		tokenStr := parts[1]

		// 3. Rozszyfruj JWT
		payload, err := security.ParseJWT(tokenStr, accessSecret)
		if err != nil {
			log.WarnMap("JWT invalid", map[string]string{
				"error": err.Error(),
				"path":  path,
			})
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		// 4. Pobierz sessionID z payload
		sessionID, ok := payload["sid"].(string)
		if !ok || sessionID == "" {
			log.WarnMap("Missing session_id in token", map[string]string{
				"payload": fmt.Sprintf("%v", payload),
				"path":    path,
			})
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing session_id in token"})
		}

		// 5. Pobierz userID z Redis po sessionID
		ctx := context.Background()
		userID, err := rdb.Get(ctx, "session:"+sessionID).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				log.WarnMap("Session not found or expired", map[string]string{
					"sessionID": sessionID,
					"path":      path,
				})
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired session"})
			}
			log.ErrorObj("Redis error", err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		// 6. Zapisz userID w ctx dla downstream
		c.Locals("userID", userID)
		log.DebugMap("User session verified", map[string]string{
			"userID": userID,
			"path":   path,
		})

		return c.Next()
	}
}
