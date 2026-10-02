package dto

import "encoding/json"

type DocumentResponse struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	DocumentType   string          `json:"document_type"`
	DocumentNumber string          `json:"document_number"`
	Status         string          `json:"status"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	IssuedAt       string          `json:"issued_at,omitempty"`
	ExpiresAt      string          `json:"expires_at,omitempty"`
	CreatedAt      string          `json:"created_at,omitempty"`
	UpdatedAt      string          `json:"updated_at,omitempty"`
}

// Deprecated: kept for compatibility with older mapper usage.
type UserDocumentResponse = DocumentResponse
