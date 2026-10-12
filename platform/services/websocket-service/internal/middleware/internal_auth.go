package middleware

import (
	"encoding/base64"

	fiber "github.com/gofiber/fiber/v3"
	"github.com/zerodayz7/platform/pkg/constants"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
)

func InternalAuthMiddleware(keyStore *httpserver.KeyStore) fiber.Handler {
	return func(c fiber.Ctx) error {
		log := shared.GetLogger()

		encodedCtx := c.Get(constants.HeaderInternalContext)
		signature := c.Get(constants.HeaderInternalSignature)
		if encodedCtx == "" || signature == "" {
			return c.Next()
		}

		senderID := c.Get("X-Internal-Service")
		if senderID == "" {
			senderID = "gateway"
		}

		hmacKey, _, ok := keyStore.GetKey(senderID)
		if !ok {
			log.Warn("Nie znaleziono klucza HMAC dla nadawcy", "sender", senderID)
			return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
		}

		payload, err := base64.StdEncoding.DecodeString(encodedCtx)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("invalid internal context")
		}

		if !reqctx.VerifyHMAC(payload, signature, hmacKey) {
			return c.Status(fiber.StatusUnauthorized).SendString("invalid internal signature")
		}

		ctxData, err := reqctx.Decode(payload)
		if err != nil {
			log.Error("Context decoding failed", "error", err, "raw_payload", encodedCtx)
			return c.Status(fiber.StatusUnauthorized).SendString("invalid context")
		}

		c.Locals(reqctx.FiberRequestContextKey, ctxData)
		return c.Next()
	}
}
