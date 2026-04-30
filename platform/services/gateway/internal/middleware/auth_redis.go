package middleware

import (
	"encoding/json"
	"errors"
	"slices"

	"github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/zerodayz7/platform/pkg/constants"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
)

type UserSession struct {
	UserID      string `json:"user_id"`
	Fingerprint string `json:"fingerprint"`
}

func AuthRedisMiddleware(rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()
		path := c.Path()

		if slices.Contains(constants.PublicPaths, path) {
			return c.Next()
		}

		clientFingerprint := c.Get(constants.HeaderDeviceFingerprint)
		if clientFingerprint == "" {
			log.Warn("Missing X-Device-Fingerprint header")
			return apperr.SendAppError(c, apperr.ErrInvalidDeviceFingerprint)
		}

		jwtPayload := c.Locals("user")
		if jwtPayload == nil {
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}
		jwtToken := jwtPayload.(*jwt.Token)
		claims := jwtToken.Claims.(jwt.MapClaims)
		sessionID, _ := claims["sid"].(string)

		redisPrefix := "session:"
		if path == "/auth/register-device" {
			redisPrefix = "setup:session:"
		}

		ctx := c.Context()
		jsonData, err := rdb.Get(ctx, redisPrefix+sessionID).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {

				log.WarnMap("Session not found", map[string]any{"sid": sessionID, "path": path})
				return apperr.SendAppError(c, apperr.ErrSessionExpired)
			}
			return apperr.SendAppError(c, apperr.ErrInternal)
		}

		var session UserSession
		if err := json.Unmarshal([]byte(jsonData), &session); err != nil {
			return apperr.SendAppError(c, apperr.ErrInternal)
		}

		if session.Fingerprint != clientFingerprint {
			log.WarnMap("Fingerprint mismatch!", map[string]any{
				"expected": session.Fingerprint,
				"received": clientFingerprint,
			})

			return apperr.SendAppError(c, apperr.ErrUntrustedDevice)
		}

		c.Locals("userID", session.UserID)
		c.Locals("sessionID", sessionID)
		c.Locals("deviceID", session.Fingerprint)

		return c.Next()
	}
}
