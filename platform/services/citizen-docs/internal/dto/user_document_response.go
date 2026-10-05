package dto

type DocumentResponse struct {
	ID               string         `json:"id"`
	UserID           string         `json:"user_id"`
	TypeCode         string         `json:"type_code,omitempty"`
	DocumentType     string         `json:"document_type,omitempty"`
	Status           string         `json:"status"`
	DocumentHash     string         `json:"document_number_hash,omitempty"`
	DocumentNumber   string         `json:"document_number,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	IssuedAt         string         `json:"issued_at,omitempty"`
	ExpiresAt        string         `json:"expires_at,omitempty"`
	CreatedAt        string         `json:"created_at,omitempty"`
	UpdatedAt        string         `json:"updated_at,omitempty"`
	IssuerSignature  string         `json:"issuer_signature,omitempty"`
	SigningKeyID     string         `json:"signing_key_id,omitempty"`
	RevocationSerial string         `json:"revocation_serial,omitempty"`
	Version          uint64         `json:"version,omitempty"`
}

// Deprecated: kept for compatibility with older mapper usage.
type UserDocumentResponse = DocumentResponse
