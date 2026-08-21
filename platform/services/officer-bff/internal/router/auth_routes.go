package router

import (
	"net/http"

	"github.com/zerodayz7/services/officer-bff/internal/di"
)

func registerAuthRoutes(mux *http.ServeMux, c *di.Container) {
	// Krok 1: Przelotówka (zwraca challenge i userId, bez tokenów)
	loginStep1Proxy, err := NewReverseProxy(c.Config.AuthServiceURL, "/auth/login", c.KeyStore)
	if err != nil {
		panic(err)
	}

	// Krok 2: Pr przechwytuje tokeny JWT i wrzuca do HttpOnly Cookies
	loginStep2Proxy, err := NewAuthTokenProxy(c.Config.AuthServiceURL, "/auth/login/step2", c.KeyStore)
	if err != nil {
		panic(err)
	}

	// Refresh: Też używa NewAuthTokenProxy bo zwraca nowe ciasteczko access_token
	refreshProxy, err := NewAuthTokenProxy(c.Config.AuthServiceURL, "/auth/refresh", c.KeyStore)
	if err != nil {
		panic(err)
	}

	logoutProxy, err := NewAuthLogoutProxy(c.Config.AuthServiceURL, "/auth/logout", c.KeyStore)
	if err != nil {
		panic(err)
	}

	meProxy, err := NewReverseProxy(c.Config.AuthServiceURL, "/auth/me", c.KeyStore)
	if err != nil {
		panic(err)
	}

	mux.HandleFunc("POST /api/v1/official/auth/login", loginStep1Proxy)
	mux.HandleFunc("POST /api/v1/official/auth/login/step2", loginStep2Proxy)
	mux.HandleFunc("POST /api/v1/official/auth/refresh", refreshProxy)
	mux.HandleFunc("POST /api/v1/official/auth/logout", logoutProxy)
	mux.HandleFunc("GET /api/v1/official/auth/me", meProxy)
}
