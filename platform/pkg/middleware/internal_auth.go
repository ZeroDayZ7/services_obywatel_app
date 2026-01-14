package middleware

import (
	"encoding/base64"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/constants"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
)

func InternalAuthMiddleware(hmacSecret []byte) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := shared.GetLogger()
		encodedCtx := c.Get(constants.HeaderInternalContext)
		signature := c.Get(constants.HeaderInternalSignature)

		if encodedCtx == "" || signature == "" {
			return c.Next()
		}

		// 2. Dekoduj base64
		payload, err := base64.StdEncoding.DecodeString(encodedCtx)
		if err != nil {
			return apperr.SendAppError(c, apperr.ErrInternalContextEncoding)
		}

		// 3. Weryfikuj podpis
		if !reqctx.Verify(payload, signature, hmacSecret) {
			return apperr.SendAppError(c, apperr.ErrInternalInvalidSignature)
		}

		// 4. Deserializuj do struktury
		ctx, err := reqctx.Decode(payload)
		if err != nil {
			log.Error("Context decoding failed",
				"error", err,
				"raw_payload", base64.StdEncoding.EncodeToString(payload),
			)
			return apperr.SendAppError(c, apperr.ErrInternalContextCorruption)
		}

		// WYSYP CAŁOŚCI
		log.DebugInfo("Context Dump", ctx)
		// 5. Wrzuć do Locals, żeby Handler go widział
		c.Locals(reqctx.FiberRequestContextKey, ctx)

		return c.Next()
	}
}
