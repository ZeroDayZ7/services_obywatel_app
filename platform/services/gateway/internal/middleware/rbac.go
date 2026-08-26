package middleware

import (
	"slices"

	"github.com/gofiber/fiber/v2"

	"github.com/zerodayz7/platform/pkg/constants" // <--- Importujesz własne stałe
	appcontext "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
)

// RequirePermissions weryfikuje, czy użytkownik posiada wszystkie wymagane uprawnienia.
//#region RequirePermissions
func RequirePermissions(requiredPermissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()
		path := c.Path()
		method := c.Method()

		ctx, ok := c.Locals("requestContext").(*appcontext.RequestContext)
		if !ok || ctx == nil || ctx.UserID == nil {
			log.Warn("RBAC Failure: Request context missing or corrupt",
				"path", path,
				"method", method,
			)

			return apperr.SendAppError(c, apperr.ErrUnauthorized)
		}

		// Superuser "root" omija sprawdzanie granulacji uprawnień (dev / god mode)
		if ctx.Role == "root" {
			c.Request().Header.Set(constants.HeaderUserID, ctx.UserID.String())
			return c.Next()
		}

		if !hasAllPermissions(ctx.Permissions, requiredPermissions) {
			log.Warn("Forbidden access attempt: insufficient permissions",
				"user_id", ctx.UserID.String(),
				"required", requiredPermissions,
				"current_role", ctx.Role,
				"user_permissions", ctx.Permissions,
				"path", path,
				"method", method,
			)

			return apperr.SendAppError(c, apperr.ErrForbidden)
		}

		c.Request().Header.Set(constants.HeaderUserID, ctx.UserID.String())

		return c.Next()
	}
}

// hasAllPermissions sprawdza czy userPerms zawiera KAŻDE z requiredPerms
//#region hasAllPermissions
func hasAllPermissions(userPerms []string, requiredPerms []string) bool {
	for _, req := range requiredPerms {
		if !slices.Contains(userPerms, req) {
			return false
		}
	}
	return true
}
