package router

import (
	"net/http"

	"github.com/zerodayz7/services/identity-service/internal/di"
)

func NewRouter(c *di.Container) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", c.HealthHandler.Check)

	// Citizen API (Go 1.22+ Verb + Path Matching)
	mux.HandleFunc("POST /api/v1/citizens", c.CitizenHandler.Register)
	mux.HandleFunc("GET /api/v1/citizens/{user_id}", c.CitizenHandler.GetByID)

	return mux
}