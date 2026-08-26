package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/zerodayz7/platform/pkg/constants"
	appcontext "github.com/zerodayz7/platform/pkg/context"
	"github.com/zerodayz7/platform/pkg/crypto"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	rdy "github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

// #region ContextBuilder
func ContextBuilder(container *di.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()

		requestID, _ := c.Locals("requestid").(string)

		reqCtx := &appcontext.RequestContext{
			IP:        c.IP(),
			RequestID: requestID,
		}

		// Odczytujemy sesje ustawione wcześniej w AuthRedisMiddleware
		userLocal := c.Locals("userSession")
		setupLocal := c.Locals("setupSession")

		// 1. ŚCIEŻKI PUBLICZNE (Brak jakiejkolwiek sesji w c.Locals)
		if userLocal == nil && setupLocal == nil {
			rawFP := c.Get(constants.HeaderDeviceFingerprint)
			if rawFP != "" {
				reqCtx.DeviceID = crypto.HashSHA256(rawFP)
			}
			c.Locals("requestContext", reqCtx)
			return c.Next()
		}

		// 2. ŚCIEŻKI CHRONIONE (Mamy sessionID z JWT)
		if sessionID, ok := c.Locals("sessionID").(uuid.UUID); ok && sessionID != uuid.Nil {
			reqCtx.SessionID = &sessionID
		} else {
			log.ErrorObj("[ContextBuilder] Missing or invalid sessionID in context", map[string]any{
				"sid": c.Locals("sessionID"),
			})
			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		// A. Obsługa UserSession (pełna sesja po zalogowaniu)
		if sessionData, ok := userLocal.(*rdy.UserSession); ok && sessionData != nil {
			reqCtx.DeviceID = sessionData.Fingerprint

			if parsedUserID, err := uuid.Parse(sessionData.UserID); err == nil {
				reqCtx.UserID = &parsedUserID
			} else {
				log.ErrorObj("[ContextBuilder] Invalid UserID string in UserSession", map[string]any{"rawUserID": sessionData.UserID})
				return apperr.SendAppError(c, apperr.ErrUnauthorized)
			}

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
		}

		// B. Obsługa SetupSession (sesja wstępna, np. /login/step2)
		if setupData, ok := setupLocal.(*rdy.SetupSession); ok && setupData != nil {
			reqCtx.DeviceID = setupData.Fingerprint

			if parsedUserID, err := uuid.Parse(setupData.UserID); err == nil {
				reqCtx.UserID = &parsedUserID
			} else {
				log.ErrorObj("[ContextBuilder] Invalid UserID string in SetupSession", map[string]any{"rawUserID": setupData.UserID})
				return apperr.SendAppError(c, apperr.ErrUnauthorized)
			}

			reqCtx.Role = setupData.Role
		}

		c.Locals("requestContext", reqCtx)
		return c.Next()
	}
}
