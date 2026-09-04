package handler

import (
	"document-renderer/internal/model"
	"document-renderer/internal/service"
	"encoding/json"
	"io"
	"net/http"

	"github.com/zerodayz7/platform/pkg/shared"
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
	log := shared.GetLogger()

	if h.maxRequestBodyBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, h.maxRequestBodyBytes)
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req model.RenderDocumentRequest
	if err := decoder.Decode(&req); err != nil {
		log.Error("Failed to decode request payload", err)
		h.writeError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid request payload structure")
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		log.Warn("Request body contains trailing bytes after JSON object")
		h.writeError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "request body must contain exactly one JSON object")
		return
	}

	if req.Template == "" {
		log.Warn("Validation failed: template field is required")
		h.writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "field 'template' is required")
		return
	}

	pdfBytes, err := h.renderService.RenderPDF(r.Context(), req.Template, req.Data, req.Options)
	if err != nil {
		log.Error("PDF generation failed", "template", req.Template, err)
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
