package router

import (
	"net/http"

	"github.com/zerodayz7/platform/services/version-service/internal/handler"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/version", handler.VersionHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}
