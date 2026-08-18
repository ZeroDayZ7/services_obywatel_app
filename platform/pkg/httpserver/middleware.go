package httpserver

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/zerodayz7/platform/pkg/constants"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
)

// InternalAuthMiddleware weryfikuje podpis HMAC-SHA256 nagłówka kontekstu wewnętrznego.
func InternalAuthMiddleware(hmacKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := shared.GetLogger()

			encodedCtx := r.Header.Get(constants.HeaderInternalContext)
			signature := r.Header.Get(constants.HeaderInternalSignature)

			if encodedCtx == "" || signature == "" {
				next.ServeHTTP(w, r)
				return
			}

			// 1. Dekoduj surowy ładunek z Base64
			payload, err := base64.StdEncoding.DecodeString(encodedCtx)
			if err != nil {
				SendError(w, r, apperr.ErrInternalContextEncoding)
				return
			}

			// 2. Weryfikuj podpis HMAC w czasie stałym (constant-time)
			if !reqctx.VerifyHMAC(payload, signature, hmacKey) {
				SendError(w, r, apperr.ErrInternalInvalidSignature)
				return
			}

			// 3. Deserializuj ładunek do struktury kontekstu
			ctxData, err := reqctx.Decode(payload)
			if err != nil {
				log.Error("Context decoding failed",
					"error", err,
					"raw_payload", encodedCtx,
				)
				SendError(w, r, apperr.ErrInternalContextCorruption)
				return
			}

			log.DebugInfo("Context Dump", ctxData)

			// 4. Zapisz zdekodowany kontekst do r.Context()
			ctx := context.WithValue(r.Context(), reqctx.RequestContextKey, ctxData)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
