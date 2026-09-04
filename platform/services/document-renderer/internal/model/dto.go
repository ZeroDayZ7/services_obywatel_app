package model

import "document-renderer/internal/renderer"

type RenderDocumentRequest struct {
	Template string               `json:"template"`
	Data     map[string]any       `json:"data"`
	Options  *renderer.PDFOptions `json:"options,omitempty"`
}
