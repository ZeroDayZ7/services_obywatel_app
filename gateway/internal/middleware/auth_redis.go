package middleware

import (
	"context"
	"errors"

	"slices"

	"github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/zerodayz7/http-server/internal/shared/logger"
)

// PublicPathsRedis - ścieżki, które NIE wymagają weryfikacji Redis
var PublicPathsRedis = []string{
	"/auth/login",
	"/auth/register",
	"/auth/refresh",
	"/auth/2fa-verify",
	"/health",
}

// AuthRedisMiddleware - middleware do weryfikacji sesji w Redis
func AuthRedisMiddleware(rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := logger.GetLogger()
		path := c.Path()

		// 1. Sprawdź, czy ścieżka jest publiczna
		if slices.Contains(PublicPathsRedis, path) {
			log.DebugMap("Public path, skipping Redis check", map[string]string{"path": path})
			return c.Next()
		}

		// 2. Pobierz JWT z ctx ustawionego przez JWT middleware
		jwtPayload := c.Locals("user")
		if jwtPayload == nil {
			log.WarnMap("JWT payload missing", map[string]string{"path": path})
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}

		// 3. Rzutowanie na *jwt.Token
		jwtToken, ok := jwtPayload.(*jwt.Token)
		if !ok {
			log.WarnMap("JWT payload has invalid type", map[string]string{"path": path})
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token payload"})
		}

		// 4. Wyciągnięcie claims jako MapClaims
		claims, ok := jwtToken.Claims.(jwt.MapClaims)
		if !ok {
			log.WarnMap("JWT claims invalid type", map[string]string{"path": path})
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token claims"})
		}

		// 5. Wyciągnij sessionID z claims
		sessionID, ok := claims["sid"].(string)
		if !ok || sessionID == "" {
			log.WarnMap("Missing session_id in JWT claims", map[string]string{"path": path})
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing session_id in token"})
		}

		// 6. Pobierz userID z Redis po sessionID
		ctx := context.Background()
		userID, err := rdb.Get(ctx, "session:"+sessionID).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				log.WarnMap("Session not found or expired", map[string]string{"sessionID": sessionID, "path": path})
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired session"})
			}
			log.ErrorObj("Redis error", err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		// 7. Zapisz userID w ctx dla downstream (np. proxy)
		c.Locals("userID", userID)
		log.DebugMap("User session verified", map[string]string{"userID": userID, "path": path})

		return c.Next()
	}
}
