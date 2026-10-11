package middleware

import (
	"github.com/gofiber/fiber/v2"
	reqctx "github.com/zerodayz7/platform/pkg/context"
)

// OperationIdMiddleware picks up X-Operation-Id header and stores it in RequestContext
func OperationIdMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		op := c.Get("X-Operation-Id")
		if op == "" {
			return c.Next()
		}

		if val := c.Locals(reqctx.FiberRequestContextKey); val != nil {
			if rc, ok := val.(*reqctx.RequestContext); ok && rc != nil {
				rc.OperationID = op
				c.Locals(reqctx.FiberRequestContextKey, rc)
			}
		}

		return c.Next()
	}
}
