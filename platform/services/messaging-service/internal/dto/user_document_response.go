package dto

type UserDocumentResponse struct {
	ID               string `json:"id"`
	TypeCode         string `json:"type_code"`
	Status           string `json:"status"`
	EncryptedMeta    string `json:"encrypted_meta"`
	EncryptedFront   string `json:"encrypted_front,omitempty"`
	EncryptedBack    string `json:"encrypted_back,omitempty"`
	IssuerSignature  string `json:"issuer_signature"`
	SigningKeyID     string `json:"signing_key_id"`
	RevocationSerial string `json:"revocation_serial"`
	Version          uint64 `json:"version"`
	IssuedAt         string `json:"issued_at,omitempty"`
	ExpiresAt        string `json:"expires_at,omitempty"`
}

type UserDocumentsResponse struct {
	Count int                    `json:"count"`
	Docs  []UserDocumentResponse `json:"docs"`
}
