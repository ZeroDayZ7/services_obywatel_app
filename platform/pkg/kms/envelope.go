package kms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ============================================================================
// CONSTANTS & TYPES
// ============================================================================

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
// EncryptDEK wysyła surowy klucz DEK z pamięci do KMS w celu zaszyfrowania go KEK-iem.
func EncryptDEK(ctx context.Context, cfg Config, keyAlias string, plaintextDEK []byte) ([]byte, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	reqBody := encryptRequest{
		KeyAlias:  keyAlias,
		Plaintext: plaintextDEK,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to marshal encrypt request: %w", err)
	}

	path := EncryptEndpoint
	url := fmt.Sprintf("%s%s", cfg.Endpoint, path)

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("kms: failed to create encrypt request: %w", err)
	}

	signAndSetHeaders(req, http.MethodPost, path, cfg)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kms: encrypt request failed: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to read encrypt response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kms: unexpected status %d on encrypt, body: %s", res.StatusCode, string(bodyBytes))
	}

	var out encryptResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to unmarshal encrypt response: %w", err)
	}

	return out.Ciphertext, nil
}
// #endregion

// #region DecryptDEK
// DecryptDEK wysyła zaszyfrowany DEK do KMS w celu odszyfrowania go KEK-iem.
func DecryptDEK(ctx context.Context, cfg Config, keyAlias string, encryptedDEK []byte) ([]byte, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	reqBody := decryptRequest{
		KeyAlias:   keyAlias,
		Ciphertext: encryptedDEK,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to marshal decrypt request: %w", err)
	}

	path := DecryptEndpoint
	url := fmt.Sprintf("%s%s", cfg.Endpoint, path)

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("kms: failed to create decrypt request: %w", err)
	}

	signAndSetHeaders(req, http.MethodPost, path, cfg)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kms: decrypt request failed: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to read decrypt response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kms: unexpected status %d on decrypt, body: %s", res.StatusCode, string(bodyBytes))
	}

	var out decryptResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to unmarshal decrypt response: %w", err)
	}

	return out.Plaintext, nil
}
// #endregion