package middleware

import (
	"github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/zerodayz7/platform/pkg/constants"
	appcontext "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

func ContextBuilder(container *di.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()

		log.DebugObj("[ContextBuilder] Building base request context", map[string]any{
			"path":   c.Path(),
			"method": c.Method(),
		})

		fingerprint := c.Get(constants.HeaderDeviceFingerprint)
		requestID, _ := c.Locals("requestid").(string)

		reqCtx := &appcontext.RequestContext{
			DeviceID:  fingerprint,
			IP:        c.IP(),
			RequestID: requestID,
		}

		userLocal := c.Locals("user")
		if userLocal == nil {
			log.DebugObj("[ContextBuilder] No 'user' in c.Locals, passing anonymous context", map[string]any{
				"path":        c.Path(),
				"fingerprint": fingerprint,
			})

			c.Locals("requestContext", reqCtx)
			return c.Next()
		}

		token, ok := userLocal.(*jwt.Token)
		if !ok || token == nil {
			log.WarnObj("[ContextBuilder] Failed to cast 'user' local to *jwt.Token", map[string]any{
				"type": userLocal,
			})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.WarnObj("[ContextBuilder] Failed to parse JWT claims to MapClaims", map[string]any{
				"rawClaims": token.Claims,
			})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		sid, _ := claims["sid"].(string)
		if sid == "" {
			log.WarnObj("[ContextBuilder] Missing or empty 'sid' claim in token", map[string]any{
				"claims": claims,
			})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		scope, _ := claims["scope"].(string)
		path := c.Path()

		var sessionData *redis.UserSession
		var err error

		if scope == "device_verify" || path == "/auth/register-device" {
			sessionData, err = container.Cache.GetSetupSession(c.Context(), sid)
		} else {
			sessionData, err = container.Cache.GetSession(c.Context(), sid)
		}

		if err != nil {
			log.WarnObj("[ContextBuilder] Session missing, invalid or expired in Redis", map[string]any{
				"sid":  sid,
				"path": path,
				"err":  err.Error(),
			})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		parsedUserID, err := uuid.Parse(sessionData.UserID)
		if err != nil {
			log.ErrorObj("[ContextBuilder] Failed to parse UserID string to UUID", map[string]any{
				"rawUserID": sessionData.UserID,
				"err":       err.Error(),
			})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		reqCtx.UserID = &parsedUserID
		reqCtx.SessionID = sid
		reqCtx.Role = sessionData.Role
		reqCtx.Permissions = sessionData.Permissions
		reqCtx.Username = sessionData.Username

		if sessionData.InstitutionID != "" {
			if parsedInst, err := uuid.Parse(sessionData.InstitutionID); err == nil {
				reqCtx.InstitutionID = &parsedInst
			}
		}

		if sessionData.DepartmentID != "" {
			if parsedDept, err := uuid.Parse(sessionData.DepartmentID); err == nil {
				reqCtx.DepartmentID = &parsedDept
			}
		}

		c.Locals("requestContext", reqCtx)

		log.DebugObj("[ContextBuilder] Authenticated context attached successfully", map[string]any{
			"userID":      parsedUserID.String(),
			"sessionID":   sid,
			"role":        sessionData.Role,
			"fingerprint": fingerprint,
		})

		return c.Next()
	}
}
