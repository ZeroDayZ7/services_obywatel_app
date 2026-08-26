// cmdr: cmdr: context\fiber.go

package context

import "github.com/gofiber/fiber/v2"

//#region MustFromFiber
func MustFromFiber(c *fiber.Ctx) *RequestContext {
	ctx, ok := c.Locals(FiberRequestContextKey).(*RequestContext)
	if !ok || ctx == nil {
		panic("request context missing or invalid")
	}
	return ctx
}
