package shared

import (
	"github.com/gofiber/fiber/v2"
)

// RequestLoggerMiddleware — loguje body i nagłówki w osobnych liniach
func RequestLoggerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := GetLogger()

		// Tworzymy mapę nagłówków w formacie string -> []string
		headers := c.GetReqHeaders() // map[string][]string

		// Body dla metod z payload
		body := map[string]any{}
		if c.Method() == fiber.MethodPost || c.Method() == fiber.MethodPut || c.Method() == fiber.MethodPatch {
			_ = c.BodyParser(&body)
		}

		// Logujemy metodę, path i body
		log.InfoObj("Incoming request", map[string]any{
			"method": c.Method(),
			"path":   c.Path(),
			"body":   body,
		})

		// Logujemy nagłówki osobno
		for k, v := range headers {
			log.InfoObj("Header", map[string]any{
				"key":   k,
				"value": v,
			})
		}

		return c.Next()
	}
}
