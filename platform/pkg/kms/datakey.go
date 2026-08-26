package kms

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

// DataKey reprezentuje parę klucza DEK zwróconą z KMS.
type DataKey struct {
	Plaintext        []byte
	Ciphertext       []byte
	MasterKeyVersion int
}

type GenerateDataKeyRequest struct {
	KeyAlias  string `json:"key_alias"`
	Algorithm string `json:"algorithm"`
}

// GenerateDataKeyResponse mapuje różne warianty odpowiedzi z serwera KMS
type GenerateDataKeyResponse struct {
	KeyAlias         string `json:"key_alias"`
	MasterKeyVersion int    `json:"master_key_version"`
	KeyVersion       int    `json:"key_version"`

	// Warianty dla Plaintext DEK
	PlaintextB64    string `json:"plaintext_dek_b64"`
	DekB64          string `json:"dek_b64"`
	PlaintextB64Alt string `json:"plaintext_b64"`

	// Warianty dla Ciphertext / Wrapped DEK
	CiphertextB64    string `json:"ciphertext_dek_b64"`
	WrappedDekB64    string `json:"wrapped_dek_b64"`
	CiphertextB64Alt string `json:"ciphertext_b64"`
}

// GenerateDataKey prosi KMS o wygenerowanie nowego DEK wewnątrz HSM/KMS.
func GenerateDataKey(ctx context.Context, cfg Config, keyAlias string) (*DataKey, error) {
	reqBody, err := json.Marshal(GenerateDataKeyRequest{
		KeyAlias:  keyAlias,
		Algorithm: AlgorithmAES256GCM,
	})
	if err != nil {
		return nil, fmt.Errorf("kms: failed to marshal generate data key request: %w", err)
	}

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodPost, EndpointGenerateDataKey, reqBody, true)
	if err != nil {
		return nil, fmt.Errorf("kms: generate data key request failed: %w", err)
	}
	defer ZeroBytes(bodyBytes)

	var out GenerateDataKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to unmarshal generate data key response: %w", err)
	}

	// Wybieramy odpowiednie pole dla Plaintext
	rawPlaintextB64 := out.PlaintextB64
	if rawPlaintextB64 == "" {
		rawPlaintextB64 = out.DekB64
	}
	if rawPlaintextB64 == "" {
		rawPlaintextB64 = out.PlaintextB64Alt
	}

	// Wybieramy odpowiednie pole dla Ciphertext
	rawCiphertextB64 := out.CiphertextB64
	if rawCiphertextB64 == "" {
		rawCiphertextB64 = out.WrappedDekB64
	}
	if rawCiphertextB64 == "" {
		rawCiphertextB64 = out.CiphertextB64Alt
	}

	// Wersja klucza
	version := out.MasterKeyVersion
	if version == 0 && out.KeyVersion != 0 {
		version = out.KeyVersion
	}

	// Walidacja czy pole nie jest puste – zabezpieczenie przed paniką
	if rawPlaintextB64 == "" {
		return nil, fmt.Errorf("kms: received empty plaintext DEK from KMS. Raw response: %s", string(bodyBytes))
	}

	plaintext, err := base64.StdEncoding.DecodeString(rawPlaintextB64)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to decode plaintext DEK base64: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(rawCiphertextB64)
	if err != nil {
		ZeroBytes(plaintext)
		return nil, fmt.Errorf("kms: failed to decode ciphertext DEK base64: %w", err)
	}

	return &DataKey{
		Plaintext:        plaintext,
		Ciphertext:       ciphertext,
		MasterKeyVersion: version,
	}, nil
}
