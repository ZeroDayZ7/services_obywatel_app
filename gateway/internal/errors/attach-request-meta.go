package errors

import "github.com/gofiber/fiber/v2"

func AttachRequestMeta(c *fiber.Ctx, err *AppError, keysToInclude ...string) {
	if err.Meta == nil {
		err.Meta = make(map[string]any)
	}

	for _, key := range keysToInclude {
		switch key {
		case "requestID":
			if val := c.Locals("requestid"); val != nil {
				err.Meta["requestID"] = val
			}
		case "ip":
			err.Meta["ip"] = c.IP()
		case "method":
			err.Meta["method"] = c.Method()
		case "path":
			err.Meta["path"] = c.Path()
		default:

			var body map[string]any
			if parseErr := c.BodyParser(&body); parseErr == nil {
				if v, ok := body[key]; ok {
					err.Meta[key] = v
				}
			}
		}
	}
}
