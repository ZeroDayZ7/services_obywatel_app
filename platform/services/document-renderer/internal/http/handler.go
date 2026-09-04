package http

import (
	"document-renderer/internal/renderer"
	"document-renderer/internal/templates"
	"encoding/json"
	"net/http"

	"github.com/go-rod/rod"
)

type RenderDocumentRequest struct {
	Template string               `json:"template"` // np. "contracts/standard.html"
	Data     map[string]any       `json:"data"`     // Dowolne dane przekazywane do Go HTML template
	Options  *renderer.PDFOptions `json:"options,omitempty"`
}

func HandleRenderDocument(browser *rod.Browser, loader *templates.Loader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RenderDocumentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.Template == "" {
			http.Error(w, "field 'template' is required", http.StatusBadRequest)
			return
		}

		htmlBytes, err := loader.Render(req.Template, req.Data)
		if err != nil {
			http.Error(w, "template rendering error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		opts := renderer.DefaultPDFOptions()
		if req.Options != nil {
			opts = *req.Options
		}

		pdfBytes, err := renderer.RenderHTMLToPDF(r.Context(), browser, string(htmlBytes), opts)
		if err != nil {
			http.Error(w, "pdf generation error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "inline; filename=\"document.pdf\"")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdfBytes)
	}
}
