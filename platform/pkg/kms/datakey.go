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
	KeyAlias string `json:"key_alias"`
}

type GenerateDataKeyResponse struct {
	KeyAlias         string `json:"key_alias"`
	PlaintextB64     string `json:"plaintext_dek_b64"`
	CiphertextB64    string `json:"ciphertext_dek_b64"`
	MasterKeyVersion int    `json:"master_key_version"`
}

// GenerateDataKey prosi KMS o wygenerowanie nowego DEK wewnątrz HSM/KMS.
func GenerateDataKey(ctx context.Context, cfg Config, keyAlias string) (*DataKey, error) {
	reqBody, err := json.Marshal(GenerateDataKeyRequest{
		KeyAlias: keyAlias,
	})
	if err != nil {
		return nil, fmt.Errorf("kms: failed to marshal generate data key request: %w", err)
	}

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodPost, "/api/v1/keys/generate-data-key", reqBody, true)
	if err != nil {
		return nil, fmt.Errorf("kms: generate data key request failed: %w", err)
	}
	defer ZeroBytes(bodyBytes)

	var out GenerateDataKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to unmarshal generate data key response: %w", err)
	}

	plaintext, err := base64.StdEncoding.DecodeString(out.PlaintextB64)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to decode plaintext DEK base64: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(out.CiphertextB64)
	if err != nil {
		ZeroBytes(plaintext)
		return nil, fmt.Errorf("kms: failed to decode ciphertext DEK base64: %w", err)
	}

	return &DataKey{
		Plaintext:        plaintext,
		Ciphertext:       ciphertext,
		MasterKeyVersion: out.MasterKeyVersion,
	}, nil
}
