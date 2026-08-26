package router

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/identity-service/internal/di"
)

// SetupMiddleware buduje łańcuch globalnych middleware dla podanego handlera.
func SetupMiddleware(handler http.Handler, container *di.Container) http.Handler {
	logger := shared.GetLogger()

	bffSecret, _, ok := container.KeyStore.GetKey("officer-bff")
	if !ok {
		logger.Warn("⚠️ Brak klucza HMAC dla officer-bff w KeyStore identity-service!")
	}

	hmacMiddleware := httpserver.InternalAuthMiddleware(bffSecret)
	loggerMiddleware := httpserver.LoggerMiddleware(logger)

	// Nakładanie middleware: Logger (zewnętrzny) -> HMAC -> Mux
	return loggerMiddleware(hmacMiddleware(handler))
}
