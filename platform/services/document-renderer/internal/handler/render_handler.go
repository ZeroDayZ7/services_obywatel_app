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
	renderService       service.RenderService
	maxRequestBodyBytes int64
}

func NewRenderHandler(renderService service.RenderService, maxRequestBodyBytes int64) RenderHandler {
	return &renderHandler{
		renderService:       renderService,
		maxRequestBodyBytes: maxRequestBodyBytes,
	}
}

func (h *renderHandler) RenderPDF(w http.ResponseWriter, r *http.Request) {
	if h.maxRequestBodyBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, h.maxRequestBodyBytes)
	}

	var req model.RenderDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode request payload: %v", err)
		h.writeError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid request payload")
		return
	}

	if req.Template == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "field 'template' is required")
		return
	}

	pdfBytes, err := h.renderService.RenderPDF(r.Context(), req.Template, req.Data, req.Options)
	if err != nil {
		log.Printf("[ERROR] PDF generation failed for template '%s': %v", req.Template, err)
		h.writeError(w, http.StatusInternalServerError, "RENDER_FAILED", "failed to generate document")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=\"document.pdf\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

func (h *renderHandler) writeError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(model.ErrorResponse{
		Code:    code,
		Message: message,
	})
}
