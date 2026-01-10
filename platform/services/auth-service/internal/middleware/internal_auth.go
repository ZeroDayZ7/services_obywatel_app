package middleware

import (
	"encoding/base64"

	"github.com/gofiber/fiber/v2"
	reqctx "github.com/zerodayz7/platform/pkg/context"
)

func InternalAuthMiddleware(hmacSecret []byte) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Pobierz nagłówki z Gatewaya
		encodedCtx := c.Get("X-Internal-Context")
		signature := c.Get("X-Internal-Signature")

		if encodedCtx == "" || signature == "" {
			// Jeśli brak, a to nie jest ścieżka publiczna - odrzuć
			// Ale uwaga: Auth ma ścieżki publiczne (Login), więc tu decydujemy
			return c.Next()
		}

		// 2. Dekoduj base64
		payload, err := base64.StdEncoding.DecodeString(encodedCtx)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "invalid context encoding"})
		}

		// 3. Weryfikuj podpis używając tego samego SECRET co w Gateway
		// Pamiętaj, że Secret masz w config.AppConfig.Internal.HMACSecret
		if !reqctx.Verify(payload, signature, hmacSecret) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "invalid internal signature"})
		}

		// 4. Deserializuj do struktury
		ctx, err := reqctx.Decode(payload)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "context corruption"})
		}

		// 5. Wrzuć do Locals, żeby Handler go widział
		c.Locals("requestContext", ctx)

		return c.Next()
	}
}
