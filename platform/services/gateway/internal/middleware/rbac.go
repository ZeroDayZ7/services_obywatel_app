package middleware

import (
	"slices"

	"github.com/gofiber/fiber/v2"

	appcontext "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
)

// RBACRequired weryfikuje czy rola lub uprawnienia w RequestContext są wystarczające
func RBACRequired(requiredPermissionOrRole string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()
		path := c.Path()

		ctx, ok := c.Locals("requestContext").(*appcontext.RequestContext)
		if !ok || ctx == nil || ctx.UserID == nil {
			log.Warn("RBAC Failure: Request context missing or corrupt",
				"path", path,
				"method", c.Method(),
			)

			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		// Superuser "root" omija sprawdzanie uprawnień
		if ctx.Role == "root" {
			c.Request().Header.Set("X-User-ID", ctx.UserID.String())
			return c.Next()
		}

		hasAccess := ctx.Role == requiredPermissionOrRole ||
			slices.Contains(ctx.Permissions, requiredPermissionOrRole)

		if !hasAccess {
			log.Warn("Forbidden access attempt: insufficient permissions", map[string]any{
				"user_id":     ctx.UserID.String(),
				"required":    requiredPermissionOrRole,
				"currentRole": ctx.Role,
				"permissions": ctx.Permissions,
				"path":        path,
			})

			return apperr.SendAppError(c, apperr.ErrForbidden)
		}

		c.Request().Header.Set("X-User-ID", ctx.UserID.String())

		return c.Next()
	}
}
