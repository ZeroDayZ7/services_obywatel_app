package dto

type DocumentResponse struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	DocumentType string `json:"document_type"`
	Status       string `json:"status"`
	DocumentHash string `json:"document_number_hash,omitempty"`
	IssuedAt     string `json:"issued_at,omitempty"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// Deprecated: kept for compatibility with older mapper usage.
type UserDocumentResponse = DocumentResponse
