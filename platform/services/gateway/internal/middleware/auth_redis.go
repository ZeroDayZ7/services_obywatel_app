package middleware

import (
	"errors"
	"slices"

	"github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"github.com/zerodayz7/platform/pkg/constants"
	"github.com/zerodayz7/platform/pkg/crypto"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	rdy "github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/config"
)

// #region AuthRedisMiddleware
func AuthRedisMiddleware(cache *rdy.Cache) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()
		path := c.Path()

		if slices.Contains(constants.PublicPaths, path) {
			return c.Next()
		}

		// 1. Pobieramy i hashujemy fingerprint od klienta
		clientFingerprint := c.Get(constants.HeaderDeviceFingerprint)
		if clientFingerprint == "" {
			log.Warn("Missing X-Device-Fingerprint header")
			return apperr.SendAppError(c, apperr.ErrInvalidDeviceFingerprint)
		}

		hashedClientFingerprint := crypto.HashSHA256(clientFingerprint)

		// 2. Walidacja obecności i typu tokena JWT
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

		// 3. Ekstrakcja danych z claims
		claimsData, err := config.ExtractJWTClaims(claims)
		if err != nil {
			log.WarnMap("[AuthRedisMiddleware] Invalid JWT claims", map[string]any{
				"path": path,
				"err":  err.Error(),
			})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		// 4. WALIDACJA FINGERPRINTU Z JWT (Szybkie odrzucenie przed wywołaniem Redisa)
		if claimsData.FingerprintHash != "" && claimsData.FingerprintHash != hashedClientFingerprint {
			log.WarnMap("[AuthRedisMiddleware] JWT Fingerprint mismatch with client header", map[string]any{
				"expected_in_jwt": claimsData.FingerprintHash,
				"received_header": hashedClientFingerprint,
			})
			return apperr.SendAppError(c, apperr.ErrUntrustedDevice)
		}

		// 5. Sprawdzenie Scope
		isDeviceVerifyPath := slices.Contains(constants.DeviceVerifyPaths, path)
		expectedScope := constants.ScopeAccess.String()
		if isDeviceVerifyPath {
			expectedScope = constants.ScopeDeviceVerify.String()
		}

		if claimsData.Scope != expectedScope {
			log.WarnMap("[AuthRedisMiddleware] Invalid token scope for path", map[string]any{
				"path":           path,
				"expected_scope": expectedScope,
				"received_scope": claimsData.Scope,
			})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		// 6. Walidacja sesji z Redisa (weryfikacja czy sesja istnieje i czy dane w Redisie są zgodne)
		if isDeviceVerifyPath {
			setupSess, err := cache.GetSetupSession(c.Context(), claimsData.SessionID)
			if err != nil {
				if errors.Is(err, redis.Nil) {
					log.WarnMap("[AuthRedisMiddleware] Setup session key not found in Redis", map[string]any{
						"sid":  claimsData.SessionID,
						"path": path,
					})
					return apperr.SendAppError(c, apperr.ErrSessionExpired)
				}
				log.ErrorMap("[AuthRedisMiddleware] Redis command error", map[string]any{
					"sid": claimsData.SessionID,
					"err": err.Error(),
				})
				return apperr.SendAppError(c, apperr.ErrInternal)
			}

			// Sprawdzenie fingerprintu w bazie Redis
			if setupSess.Fingerprint != hashedClientFingerprint {
				log.WarnMap("[AuthRedisMiddleware] Redis Session Fingerprint mismatch", map[string]any{
					"expected_in_redis": setupSess.Fingerprint,
					"received_header":   hashedClientFingerprint,
				})
				return apperr.SendAppError(c, apperr.ErrUntrustedDevice)
			}

			// Ustawienie sessionID w kontekście
			c.Locals("setupSession", setupSess)
			c.Locals("sessionID", claimsData.SessionID)
		} else {
			userSess, err := cache.GetSession(c.Context(), claimsData.SessionID)
			if err != nil {
				if errors.Is(err, redis.Nil) {
					log.WarnMap("[AuthRedisMiddleware] User session key not found in Redis", map[string]any{
						"sid":  claimsData.SessionID,
						"path": path,
					})
					return apperr.SendAppError(c, apperr.ErrSessionExpired)
				}
				log.ErrorMap("[AuthRedisMiddleware] Redis command error", map[string]any{
					"sid": claimsData.SessionID,
					"err": err.Error(),
				})
				return apperr.SendAppError(c, apperr.ErrInternal)
			}

			// Sprawdzenie fingerprintu w bazie Redis
			if userSess.Fingerprint != hashedClientFingerprint {
				log.WarnMap("[AuthRedisMiddleware] Redis Session Fingerprint mismatch", map[string]any{
					"expected_in_redis": userSess.Fingerprint,
					"received_header":   hashedClientFingerprint,
				})
				return apperr.SendAppError(c, apperr.ErrUntrustedDevice)
			}

			c.Locals("userSession", userSess)
			c.Locals("sessionID", claimsData.SessionID)
		}

		return c.Next()

	}

}
