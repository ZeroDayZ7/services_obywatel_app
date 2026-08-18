// platform/pkg/httpserver/health.go
package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// HealthChecker reprezentuje dowolną funkcję sprawdzającą stan zależności (DB, Redis, KMS itp.)
type HealthChecker func(ctx context.Context) error

type healthResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// NewHealthHandler tworzy generyczny, uniwersalny handler HTTP dla sprawdzenia stanu usługi
func NewHealthHandler(checkers ...HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		for _, check := range checkers {
			if check == nil {
				continue
			}
			if err := check(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(healthResponse{
					Status: "UNHEALTHY",
					Error:  err.Error(),
				})
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(healthResponse{
			Status: "UP",
		})
	}
}
