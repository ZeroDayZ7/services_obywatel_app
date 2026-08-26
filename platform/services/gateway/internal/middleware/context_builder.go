package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/zerodayz7/platform/pkg/constants"
	appcontext "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	rdy "github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

// #region ContextBuilder
func ContextBuilder(container *di.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()

		fingerprint := c.Get(constants.HeaderDeviceFingerprint)
		requestID, _ := c.Locals("requestid").(string)

		reqCtx := &appcontext.RequestContext{
			DeviceID:  fingerprint,
			IP:        c.IP(),
			RequestID: requestID,
		}

		// Ścieżki publiczne (brak użyszkodnika / sesji w c.Locals)
		userLocal := c.Locals("userSession")
		if userLocal == nil {
			c.Locals("requestContext", reqCtx)
			return c.Next()
		}

		log.DebugJSON("[Context Builder]", map[string]any{
			"userSession": userLocal,
		})

		sessionData, ok := userLocal.(*rdy.UserSession)
		if !ok || sessionData == nil {
			log.ErrorObj("[ContextBuilder] Failed to cast 'userSession' from locals", map[string]any{"local": userLocal})
			return apperr.SendAppError(c, apperr.ErrInternal)
		}

		sessionID, ok := c.Locals("sessionID").(uuid.UUID)
		if !ok || sessionID == uuid.Nil {
			log.ErrorObj("[ContextBuilder] Missing or invalid sessionID in context", map[string]any{
				"sid": c.Locals("sessionID"),
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
		reqCtx.SessionID = &sessionID
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
		return c.Next()
	}
}
