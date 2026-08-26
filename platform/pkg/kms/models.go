// cmdr: kms\models.go

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

type SymmetricKeyRequest struct {
	ServiceID string `json:"service_id"`
	Algorithm string `json:"algorithm"`
}

type SymmetricKeyResponse struct {
	ServiceID string `json:"service_id"`
	Algorithm string `json:"algorithm"`
	Version   int    `json:"version"`
	KeyB64    string `json:"key_b64"`
}

type GenerateKeyRequest struct {
	ServiceID string `json:"service_id"`
	Algorithm string `json:"algorithm"`
	Purpose   string `json:"purpose"`
}
