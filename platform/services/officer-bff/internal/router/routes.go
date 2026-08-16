package router

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/officer-bff/internal/di"
)

func NewRouter(c *di.Container) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", httpserver.NewHealthHandler())

	// 1. LOGIN: Proxy do auth-service + konwersja tokenów JSON na ciasteczka
	loginProxy, err := NewAuthLoginProxy(c.Config.AuthServiceURL)
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("POST /api/v1/auth/login", loginProxy)

	// 2. LOGOUT: Zwykłe proxy przekazujące żądanie wprost do auth-service
	logoutProxy, err := NewSingleHostProxy(c.Config.AuthServiceURL)
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("POST /api/v1/auth/logout", logoutProxy)

	// 3. REJESTRACJA: Dedykowana logika BFF (wywołuje auth-service i identity-service)
	mux.HandleFunc("POST /api/v1/official/citizens/register", c.OfficialHandler.RegisterCitizen)

	// --- LOGGING MIDDLEWARE ---
	// Pobieramy instancję loggera z shared (lub c.Logger jeśli przekazujesz go w DI)
	log := shared.GetLogger()

	// Owijamy całe 'mux' naszym middleware i zwracamy powiązaną strukturę http.Handler
	return httpserver.LoggerMiddleware(log)(mux)
}
