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

	// 2. Proxies do AuthService (przekazujemy targetServiceID oraz keyStore)
	loginProxy, err := NewAuthLoginProxy(c.Config.AuthServiceURL, "/auth/login", c.KeyStore)
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("POST /api/v1/official/auth/login", loginProxy)

	logoutProxy, err := NewSingleHostProxy(c.Config.AuthServiceURL, "/auth/logout", "auth-service", c.KeyStore)
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("POST /api/v1/auth/logout", logoutProxy)

	// 3. Dedykowane handlery
	mux.HandleFunc("POST /api/v1/official/citizens/register", c.OfficialHandler.RegisterCitizen)

	// --- SETUP MIDDLEWARE ---
	log := shared.GetLogger()

	// Pobieramy klucz weryfikacji przychodzącego ruchu z Gatewaya (hmac-gateway-officer-bff)
	gwSecret, _, ok := c.KeyStore.GetKey("gateway-service")
	if !ok {
		log.Warn("⚠️ Brak klucza HMAC dla gateway-service w KeyStore!")
	}

	// Tworzymy middleware HMAC dla ruchu wchodzącego
	hmacMiddleware := httpserver.InternalAuthMiddleware(gwSecret)
	loggerMiddleware := httpserver.LoggerMiddleware(log)

	// Przeływ: Request -> Logger -> HMAC Auth -> Mux
	var handler http.Handler = mux
	handler = hmacMiddleware(handler)
	handler = loggerMiddleware(handler)

	return handler
}
