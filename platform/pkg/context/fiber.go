package context

import "github.com/gofiber/fiber/v2"

func MustFromFiber(c *fiber.Ctx) *RequestContext {
	ctx, ok := c.Locals(FiberRequestContextKey).(*RequestContext)
	if !ok || ctx == nil {
		panic("request context missing or invalid")
	}
	return ctx
}
