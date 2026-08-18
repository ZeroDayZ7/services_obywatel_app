package router

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/officer-bff/internal/di"
)

func NewRouter(c *di.Container) http.Handler {
	mux := http.NewServeMux()

	// 1. Rejestracja tras
	registerHealthRoutes(mux)
	registerAuthRoutes(mux, c)
	registerOfficialRoutes(mux, c)

	// 2. Aplikacja globalnych middleware (od zewnątrz do wewnątrz)
	return applyGlobalMiddleware(mux, c)
}

func registerHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", httpserver.NewHealthHandler())
}

func registerAuthRoutes(mux *http.ServeMux, c *di.Container) {
	loginProxy, err := NewAuthLoginProxy(c.Config.AuthServiceURL, "/auth/login", c.KeyStore)
	if err != nil {
		panic(err)
	}

	logoutProxy, err := NewAuthLogoutProxy(c.Config.AuthServiceURL, "/auth/logout", c.KeyStore)
	if err != nil {
		panic(err)
	}

	// Standardowe proxy przepuszczające ciastko do Auth Service
	meProxy, err := NewReverseProxy(c.Config.AuthServiceURL, "/auth/me", c.KeyStore)
	if err != nil {
		panic(err)
	}

	mux.HandleFunc("POST /api/v1/official/auth/login", loginProxy)
	mux.HandleFunc("POST /api/v1/official/auth/logout", logoutProxy)
	mux.HandleFunc("GET /api/v1/official/auth/me", meProxy)
}

func registerOfficialRoutes(mux *http.ServeMux, c *di.Container) {
	mux.HandleFunc("POST /api/v1/official/citizens/register", c.OfficialHandler.RegisterCitizen)
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
