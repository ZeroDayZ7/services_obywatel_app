package middleware

import (
	"errors"
	"slices"

	"github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodayz7/platform/pkg/constants"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	rdy "github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
)

//#region AuthRedisMiddleware
func AuthRedisMiddleware(cache *rdy.Cache) fiber.Handler {
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

		hashedClientFingerprint := shared.HashSHA256(clientFingerprint)

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

		tokenScope, _ := claims["scope"].(string)
		isDeviceVerifyPath := slices.Contains(constants.DeviceVerifyPaths, path)

		expectedScope := constants.ScopeAccess.String()
		if isDeviceVerifyPath {
			expectedScope = constants.ScopeDeviceVerify.String()
		}

		if tokenScope != expectedScope {
			log.WarnMap("[AuthRedisMiddleware] Invalid token scope for path", map[string]any{
				"path":           path,
				"expected_scope": expectedScope,
				"received_scope": tokenScope,
			})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		sessionIDStr, _ := claims["sid"].(string)
		sessionID, err := uuid.Parse(sessionIDStr)
		if err != nil || sessionID == uuid.Nil {
			log.WarnMap("[AuthRedisMiddleware] Invalid or missing sid claim", map[string]any{
				"path": path,
				"sid":  sessionIDStr,
			})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		// UŻYCIE GOTOWYCH METOD Z PKG/REDIS
		var session *rdy.UserSession

		if isDeviceVerifyPath {
			session, err = cache.GetSetupSession(c.Context(), sessionID)
		} else {
			session, err = cache.GetSession(c.Context(), sessionID)
		}

		if err != nil {
			if errors.Is(err, redis.Nil) {
				log.WarnMap("[AuthRedisMiddleware] Key not found in Redis", map[string]any{
					"sid":  sessionID,
					"path": path,
				})
				return apperr.SendAppError(c, apperr.ErrSessionExpired)
			}
			log.ErrorMap("[AuthRedisMiddleware] Redis command error", map[string]any{
				"sid": sessionID,
				"err": err.Error(),
			})
			return apperr.SendAppError(c, apperr.ErrInternal)
		}

		if session.Fingerprint != hashedClientFingerprint {
			log.WarnMap("[AuthRedisMiddleware] Fingerprint mismatch", map[string]any{
				"expected": session.Fingerprint,
				"received": hashedClientFingerprint,
			})
			return apperr.SendAppError(c, apperr.ErrUntrustedDevice)
		}

		// Przekazujemy sesję oraz sid z JWT dalej do ContextBuilder
		c.Locals("sessionID", sessionID)
		c.Locals("userSession", session)

		return c.Next()
	}
}
