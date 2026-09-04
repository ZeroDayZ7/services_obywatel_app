package handler

import (
	"context"
	"document-renderer/internal/renderer"
	"net/http"
	"time"

	"github.com/zerodayz7/platform/pkg/shared"
)

type HealthHandler interface {
	Live(w http.ResponseWriter, r *http.Request)
	Ready(w http.ResponseWriter, r *http.Request)
}

type healthHandler struct {
	pdfRenderer renderer.PDFRenderer
}

func NewHealthHandler(pdfRenderer renderer.PDFRenderer) HealthHandler {
	return &healthHandler{
		pdfRenderer: pdfRenderer,
	}
}

func (h *healthHandler) Live(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *healthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	log := shared.GetLogger()

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := h.pdfRenderer.Ping(ctx); err != nil {
		log.Warn("Readiness probe failed: Chromium unresponsive", "error", err)
		http.Error(w, "Service Unavailable: Chromium unresponsive", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("READY"))
}
