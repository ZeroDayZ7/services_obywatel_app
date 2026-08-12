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
	rdy "github.com/zerodayz7/platform/pkg/redis"
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
			log.WarnMap("JWT payload missing in c.Locals('user')", map[string]any{"path": path})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		jwtToken, ok := jwtPayload.(*jwt.Token)
		if !ok {
			log.ErrorMap("Failed to cast jwtPayload to *jwt.Token", map[string]any{"payload": jwtPayload})
			return apperr.SendAppError(c, apperr.ErrInternal)
		}

		claims, ok := jwtToken.Claims.(jwt.MapClaims)
		if !ok {
			log.ErrorMap("Failed to cast claims to jwt.MapClaims", map[string]any{"claims": jwtToken.Claims})
			return apperr.SendAppError(c, apperr.ErrInternal)
		}

		// 1. Walidacja Token Scope (zanim odpytamy Redisa)
		tokenScope, _ := claims["scope"].(string)

		expectedScope := constants.ScopeAccess.String()
		redisPrefix := rdy.SessionPrefix

		if path == "/auth/register-device" || path == "/auth/verify-device" {
			expectedScope = constants.ScopeDeviceVerify.String()
			redisPrefix = rdy.SetupSessionPrefix
		}

		if tokenScope != expectedScope {
			log.WarnMap("[AuthRedisMiddleware] Invalid token scope for path", map[string]any{
				"path":           path,
				"expected_scope": expectedScope,
				"received_scope": tokenScope,
			})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		sessionID, _ := claims["sid"].(string)
		fullRedisKey := rdy.SessionPrefix + sessionID

		// LOG DIAGNOSTYCZNY - Sprawdzenie dokładnego klucza i scope
		log.DebugMap("[AuthRedisMiddleware] Fetching session from Redis", map[string]any{
			"path":           path,
			"sid_from_jwt":   sessionID,
			"token_scope":    tokenScope,
			"redis_prefix":   redisPrefix,
			"full_redis_key": fullRedisKey,
		})

		ctx := c.Context()
		jsonData, err := rdb.Get(ctx, fullRedisKey).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				log.WarnMap("[AuthRedisMiddleware] Key not found in Redis", map[string]any{
					"looked_for_key": fullRedisKey,
					"sid":            sessionID,
					"path":           path,
				})
				return apperr.SendAppError(c, apperr.ErrSessionExpired)
			}
			log.ErrorMap("[AuthRedisMiddleware] Redis command error", map[string]any{
				"key": fullRedisKey,
				"err": err.Error(),
			})
			return apperr.SendAppError(c, apperr.ErrInternal)
		}

		var session rdy.UserSession
		if err := json.Unmarshal([]byte(jsonData), &session); err != nil {
			log.ErrorMap("[AuthRedisMiddleware] JSON unmarshal failed", map[string]any{
				"raw_json": jsonData,
				"err":      err.Error(),
			})
			return apperr.SendAppError(c, apperr.ErrInternal)
		}

		if session.Fingerprint != clientFingerprint {
			log.WarnMap("[AuthRedisMiddleware] Fingerprint mismatch", map[string]any{
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
