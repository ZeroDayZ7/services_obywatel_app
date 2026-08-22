package router

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/officer-bff/internal/di"
)

func NewRouter(c *di.Container) http.Handler {
	mux := http.NewServeMux()

	registerHealthRoutes(mux)
	registerAuthRoutes(mux, c)
	registerOfficialRoutes(mux, c)

	return applyGlobalMiddleware(mux, c)
}

func applyGlobalMiddleware(handler http.Handler, c *di.Container) http.Handler {
	log := shared.GetLogger()

	gwSecret, _, ok := c.KeyStore.GetKey("gateway-service")
	if !ok {
		log.Warn("⚠️ Brak klucza HMAC dla gateway-service w KeyStore!")
	}

	hmacMiddleware := httpserver.InternalAuthMiddleware(gwSecret)
	loggerMiddleware := httpserver.LoggerMiddleware(log)

	handler = hmacMiddleware(handler)
	handler = loggerMiddleware(handler)

	return handler
}
