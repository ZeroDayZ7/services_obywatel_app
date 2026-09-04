package handler

import (
	"document-renderer/internal/model"
	"document-renderer/internal/service"
	"encoding/json"
	"log"
	"net/http"
)

type RenderHandler interface {
	RenderPDF(w http.ResponseWriter, r *http.Request)
}

type renderHandler struct {
	renderService service.RenderService
}

func NewRenderHandler(renderService service.RenderService) RenderHandler {
	return &renderHandler{
		renderService: renderService,
	}
}

func (h *renderHandler) RenderPDF(w http.ResponseWriter, r *http.Request) {
	var req model.RenderDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode request payload: %v", err)
		http.Error(w, "invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Template == "" {
		log.Printf("[WARN] Validation failed: field 'template' is required")
		http.Error(w, "field 'template' is required", http.StatusBadRequest)
		return
	}

	// PODGLĄD PRZYCHODZĄCYCH DANYCH W LOGACH
	if rawData, err := json.MarshalIndent(req.Data, "", "  "); err == nil {
		log.Printf("[DEBUG] Incoming payload for template %s:\n%s", req.Template, string(rawData))
	}

	pdfBytes, err := h.renderService.RenderPDF(r.Context(), req.Template, req.Data, req.Options)
	if err != nil {
		log.Printf("[ERROR] PDF generation error (%s): %v", req.Template, err)
		http.Error(w, "pdf generation error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=\"document.pdf\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}
