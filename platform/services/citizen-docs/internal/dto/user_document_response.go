package dto

type UserDocumentResponse struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	EncryptedMeta string `json:"encrypted_meta"`
	IssuedAt      string `json:"issued_at,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

type UserDocumentsResponse struct {
	Count int                    `json:"count"`
	Docs  []UserDocumentResponse `json:"docs"`
}
