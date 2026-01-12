package middleware

import (
	"encoding/base64"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/constants"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	"github.com/zerodayz7/platform/pkg/shared"
)

func InternalAuthMiddleware(hmacSecret []byte) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()
		encodedCtx := c.Get(constants.HeaderInternalContext)
		signature := c.Get(constants.HeaderInternalSignature)

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
			log.Error("Context decoding failed",
				"error", err,
				"raw_payload", base64.StdEncoding.EncodeToString(payload),
			)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "context corruption"})
		}

		// WYSYP CAŁOŚCI (używając Twojej dedykowanej metody)
		log.DebugInfo("Context Dump", ctx)
		// 5. Wrzuć do Locals, żeby Handler go widział
		c.Locals(reqctx.FiberRequestContextKey, ctx)

		return c.Next()
	}
}
