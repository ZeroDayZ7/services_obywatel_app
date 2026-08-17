package router

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/officer-bff/internal/di"
)

func NewRouter(c *di.Container) http.Handler {
	mux := http.NewServeMux()

	// 1. Endpointy publiczne
	mux.HandleFunc("GET /health", httpserver.NewHealthHandler())

	// 2. Proxies do AuthService z obsługą Cookie Handlera
	loginProxy, err := NewAuthLoginProxy(c.Config.AuthServiceURL, "/auth/login", c.KeyStore)
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("POST /api/v1/official/auth/login", loginProxy)

	logoutProxy, err := NewAuthLogoutProxy(c.Config.AuthServiceURL, "/auth/logout", c.KeyStore)
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("POST /api/v1/official/auth/logout", logoutProxy)

	// 3. Dedykowane handlery
	mux.HandleFunc("POST /api/v1/official/citizens/register", c.OfficialHandler.RegisterCitizen)

	// --- SETUP MIDDLEWARE ---
	log := shared.GetLogger()

	gwSecret, _, ok := c.KeyStore.GetKey("gateway-service")
	if !ok {
		log.Warn("⚠️ Brak klucza HMAC dla gateway-service w KeyStore!")
	}

	hmacMiddleware := httpserver.InternalAuthMiddleware(gwSecret)
	loggerMiddleware := httpserver.LoggerMiddleware(log)

	var handler http.Handler = mux
	handler = hmacMiddleware(handler)
	handler = loggerMiddleware(handler)

	return handler
}
