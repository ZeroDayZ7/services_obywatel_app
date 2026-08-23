package kms

import "time"

type KeyResponse struct {
	ID           string    `json:"id"`
	ServiceID    string    `json:"service_id"`
	Algorithm    string    `json:"algorithm"`
	Purpose      string    `json:"purpose"`
	PublicKeyPEM string    `json:"public_key_pem,omitempty"`
	Status       string    `json:"status,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

type PrivateKeyResponse struct {
	ID            string `json:"id"`
	ServiceID     string `json:"service_id"`
	Algorithm     string `json:"algorithm"`
	PrivateKeyPEM string `json:"private_key_pem"`
}

type SymmetricKeyResponse struct {
	ID        string `json:"id"`
	ServiceID string `json:"service_id"`
	Purpose   string `json:"purpose"`
	KeyBase64 string `json:"key_base64"`
	Version   int    `json:"version"`
}

type GenerateKeyRequest struct {
	ServiceID string `json:"service_id"`
	Algorithm string `json:"algorithm"`
	Purpose   string `json:"purpose"`
}
