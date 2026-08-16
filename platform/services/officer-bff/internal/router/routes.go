package router

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/officer-bff/internal/di"
)

func NewRouter(c *di.Container) http.Handler {
	mux := http.NewServeMux()

	// 1. Endpointy publiczne (nie wymagają HMAC)
	mux.HandleFunc("GET /health", httpserver.NewHealthHandler())

	// 2. Proxies / Trasy z Gatewaya
	loginProxy, err := NewAuthLoginProxy(c.Config.AuthServiceURL)
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("POST /api/v1/auth/login", loginProxy)

	logoutProxy, err := NewSingleHostProxy(c.Config.AuthServiceURL)
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("POST /api/v1/auth/logout", logoutProxy)

	// 3. Dedykowane handlery
	mux.HandleFunc("POST /api/v1/official/citizens/register", c.OfficialHandler.RegisterCitizen)

	// --- SETUP MIDDLEWARE ---
	log := shared.GetLogger()

	// Pobieramy surowy klucz bajtowy z pobranej wcześniej konfiguracji KMS
	hmacKey := []byte(c.Config.Internal.HMACSecret)

	// Tworzymy middleware HMAC
	hmacMiddleware := httpserver.InternalAuthMiddleware(hmacKey)
	loggerMiddleware := httpserver.LoggerMiddleware(log)

	// Owijamy router: najpierw Logger, potem HMAC Auth
	// Przeływ: Request -> Logger -> HMAC Auth -> Mux (Endpoint)
	var handler http.Handler = mux
	handler = hmacMiddleware(handler)
	handler = loggerMiddleware(handler)

	return handler
}
