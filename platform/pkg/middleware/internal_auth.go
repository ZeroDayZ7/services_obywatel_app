package middleware

import (
	"encoding/base64"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/constants"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
)

// InternalAuthMiddleware weryfikuje podpis HMAC-SHA256 na podstawie nadawcy żądania.
func InternalAuthMiddleware(keyStore *httpserver.KeyStore) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()

		encodedCtx := c.Get(constants.HeaderInternalContext)
		signature := c.Get(constants.HeaderInternalSignature)

		if encodedCtx == "" || signature == "" {
			return c.Next()
		}

		// 1. Identyfikuj nadawcę żądania (domyślnie 'gateway', jeśli brak nagłówka)
		senderID := c.Get("X-Internal-Service", "gateway")

		// 2. Pobierz właściwy klucz HMAC z KeyStore
		hmacKey, _, ok := keyStore.GetKey(senderID)
		if !ok {
			log.Warn("⚠️ Nie znaleziono klucza HMAC dla nadawcy", "sender", senderID)
			return apperr.SendAppError(c, apperr.ErrInternalInvalidSignature)
		}

		// 3. Dekoduj surowy ładunek z Base64
		payload, err := base64.StdEncoding.DecodeString(encodedCtx)
		if err != nil {
			return apperr.SendAppError(c, apperr.ErrInternalContextEncoding)
		}

		// 4. Weryfikuj podpis HMAC w czasie stałym (constant-time)
		if !reqctx.VerifyHMAC(payload, signature, hmacKey) {
			return apperr.SendAppError(c, apperr.ErrInternalInvalidSignature)
		}

		// 5. Deserializuj ładunek do struktury kontekstu
		ctx, err := reqctx.Decode(payload)
		if err != nil {
			log.Error("Context decoding failed",
				"error", err,
				"raw_payload", encodedCtx,
			)
			return apperr.SendAppError(c, apperr.ErrInternalContextCorruption)
		}

		log.DebugInfo("Context Dump", ctx)
		c.Locals(reqctx.FiberRequestContextKey, ctx)

		return c.Next()
	}
}
