package kms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	EncryptEndpoint = "/api/v1/encrypt"
	DecryptEndpoint = "/api/v1/decrypt"
)

type encryptRequest struct {
	KeyAlias  string `json:"key_alias"`
	Plaintext []byte `json:"plaintext"`
}

type encryptResponse struct {
	KeyAlias   string `json:"key_alias"`
	Ciphertext []byte `json:"ciphertext"`
	KeyVersion int    `json:"key_version"`
}

type decryptRequest struct {
	KeyAlias   string `json:"key_alias"`
	Ciphertext []byte `json:"ciphertext"`
}

type decryptResponse struct {
	Plaintext []byte `json:"plaintext"`
}

// #region EncryptDEK
func EncryptDEK(ctx context.Context, cfg Config, keyAlias string, plaintextDEK []byte) ([]byte, error) {
	reqBody, err := json.Marshal(encryptRequest{
		KeyAlias:  keyAlias,
		Plaintext: plaintextDEK,
	})
	if err != nil {
		return nil, fmt.Errorf("kms: failed to marshal encrypt request: %w", err)
	}

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodPost, EncryptEndpoint, reqBody, true)
	if err != nil {
		return nil, fmt.Errorf("kms: encrypt request failed: %w", err)
	}

	var out encryptResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to unmarshal encrypt response: %w", err)
	}

	return out.Ciphertext, nil
}

// #region DecryptDEK
func DecryptDEK(ctx context.Context, cfg Config, keyAlias string, encryptedDEK []byte) ([]byte, error) {
	reqBody, err := json.Marshal(decryptRequest{
		KeyAlias:   keyAlias,
		Ciphertext: encryptedDEK,
	})
	if err != nil {
		return nil, fmt.Errorf("kms: failed to marshal decrypt request: %w", err)
	}

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodPost, DecryptEndpoint, reqBody, true)
	if err != nil {
		return nil, fmt.Errorf("kms: decrypt request failed: %w", err)
	}

	var out decryptResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to unmarshal decrypt response: %w", err)
	}

	return out.Plaintext, nil
}
