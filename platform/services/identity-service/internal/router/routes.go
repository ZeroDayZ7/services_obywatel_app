package router

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/services/identity-service/internal/di"
)

func NewRouter(c *di.Container) http.Handler {
	mux := http.NewServeMux()

	// Health check weryfikujący połączenie z bazą przez c.DB.Ping
	mux.HandleFunc("GET /health", httpserver.NewHealthHandler(c.DB.Ping))

	// Citizen API
	mux.HandleFunc("POST /api/v1/citizens", c.CitizenHandler.Register)
	mux.HandleFunc("GET /api/v1/citizens/{user_id}", c.CitizenHandler.GetByID)

	return mux
}
