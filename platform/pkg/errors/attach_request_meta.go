package errors

import (
	"maps"

	"github.com/gofiber/fiber/v2"
)

func AttachRequestMeta(c *fiber.Ctx, err *AppError, keysToInclude ...string) {
	// KLUCZOWE: Tworzymy kopię mapy, aby nie modyfikować globalnego błędu
	newMeta := maps.Clone(err.Meta)
	if newMeta == nil {
		newMeta = make(map[string]any)
	}

	for _, key := range keysToInclude {
		switch key {
		case "requestID":
			if val := c.Locals("requestid"); val != nil {
				newMeta["requestID"] = val
			}
		case "ip":
			newMeta["ip"] = c.IP()
		case "method":
			newMeta["method"] = c.Method()
		case "path":
			newMeta["path"] = c.Path()
		default:
			var body map[string]any
			if parseErr := c.BodyParser(&body); parseErr == nil {
				if v, ok := body[key]; ok {
					newMeta[key] = v
				}
			}
		}
	}
	err.Meta = newMeta
}
