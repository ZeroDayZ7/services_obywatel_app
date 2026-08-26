package router

import (
	"net/http"

	"github.com/zerodayz7/services/officer-bff/internal/di"
)

//#region registerAuthRoutes
func registerAuthRoutes(mux *http.ServeMux, c *di.Container) {
	loginStep1Proxy, err := NewReverseProxy(c.Config.AuthServiceURL, "/auth/login", c.KeyStore)
	if err != nil {
		panic(err)
	}

	loginStep2Proxy, err := NewAuthTokenProxy(
		c.Config.AuthServiceURL,
		"/auth/login/step2",
		c.KeyStore,
		c.Config.AccessTokenTTL,
		c.Config.RefreshTokenTTL,
	)
	if err != nil {
		panic(err)
	}

	refreshProxy, err := NewAuthTokenProxy(
		c.Config.AuthServiceURL,
		"/auth/refresh",
		c.KeyStore,
		c.Config.AccessTokenTTL,
		c.Config.RefreshTokenTTL,
	)
	if err != nil {
		panic(err)
	}

	logoutProxy, err := NewAuthLogoutProxy(c.Config.AuthServiceURL, "/auth/logout", c.KeyStore)
	if err != nil {
		panic(err)
	}

	mux.HandleFunc("POST /api/v1/official/auth/login", loginStep1Proxy)
	mux.HandleFunc("POST /api/v1/official/auth/login/step2", loginStep2Proxy)
	mux.HandleFunc("POST /api/v1/official/auth/refresh", refreshProxy)
	mux.HandleFunc("POST /api/v1/official/auth/logout", logoutProxy)
}
