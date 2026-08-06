package kms

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// ============================================================================
// CONSTANTS
// ============================================================================

const (
	DefaultTimeout = 5 * time.Second

	PrivateKeyEndpoint = "/api/v1/keys/private"
	PublicKeyEndpoint  = "/api/v1/keys/public/%s/%s"

	HeaderServiceName   = "X-Service-Name"
	HeaderTimestamp     = "X-Timestamp"
	HeaderHMACSignature = "X-HMAC-Signature"
	HeaderAccept        = "Accept"
	HeaderContentType   = "Content-Type"

	MIMEApplicationJSON = "application/json"

	DefaultAlgorithm = "Ed25519"
)

// ============================================================================
// TYPES
// ============================================================================

type Config struct {
	Endpoint      string        `json:"endpoint"`
	ServiceName   string        `json:"service_name"`
	ServiceSecret string        `json:"service_secret"`
	Timeout       time.Duration `json:"timeout"`
}

type getPrivateKeyRequest struct {
	ServiceID string `json:"service_id"`
	Algorithm string `json:"algorithm"`
}

type privateKeyResponse struct {
	ServiceID       string `json:"service_id"`
	Algorithm       string `json:"algorithm"`
	Version         int    `json:"version"`
	PrivateKeyBytes []byte `json:"private_key_bytes"`
}

type publicKeyResponse struct {
	ServiceID      string `json:"service_id"`
	Algorithm      string `json:"algorithm"`
	Version        int    `json:"version"`
	PublicKeyBytes []byte `json:"public_key_bytes"`
}

// #region FetchAuthPrivateKey
// FetchAuthPrivateKey pobiera klucz prywatny dla bieżącego serwisu (używając HTTP POST)
func FetchAuthPrivateKey(ctx context.Context, cfg Config) (ed25519.PrivateKey, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	reqBody := getPrivateKeyRequest{
		ServiceID: cfg.ServiceName,
		Algorithm: DefaultAlgorithm,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to marshal request body: %w", err)
	}

	path := PrivateKeyEndpoint
	url := fmt.Sprintf("%s%s", cfg.Endpoint, path)

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("kms: failed to create request: %w", err)
	}

	signAndSetHeaders(req, http.MethodPost, path, cfg)

	log.Printf("[KMS-CLIENT] Wysyłanie żądania POST -> %s", url)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kms: request execution failed: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to read response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kms: unexpected status code %d fetching private key, body: %s", res.StatusCode, string(bodyBytes))
	}

	var out privateKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to decode response JSON: %w", err)
	}

	if len(out.PrivateKeyBytes) == 0 {
		return nil, fmt.Errorf("kms: private_key_bytes is empty in KMS response payload")
	}

	var privKey ed25519.PrivateKey

	// Ed25519: Jeśli z KMS dostajemy 32 bajty (seed), zamieniamy go na pełny klucz prywatny (64 bajty)
	if len(out.PrivateKeyBytes) == ed25519.SeedSize {
		privKey = ed25519.NewKeyFromSeed(out.PrivateKeyBytes)
	} else if len(out.PrivateKeyBytes) == ed25519.PrivateKeySize {
		privKey = ed25519.PrivateKey(out.PrivateKeyBytes)
	} else {
		return nil, fmt.Errorf("kms: invalid private key bytes length: %d (expected 32 or 64)", len(out.PrivateKeyBytes))
	}

	log.Printf("[KMS-CLIENT] ✅ Sukces! Pobrano klucz prywatny Ed25519 (wersja %d, rozmiarem %d bajtów)", out.Version, len(privKey))
	return privKey, nil
}

// #endregion

// #region FetchPublicKey
// FetchPublicKey pobiera klucz publiczny wskazanego serwisu (używając HTTP GET)
func FetchPublicKey(ctx context.Context, cfg Config, targetService string) (ed25519.PublicKey, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	path := fmt.Sprintf(PublicKeyEndpoint, targetService, DefaultAlgorithm)
	url := fmt.Sprintf("%s%s", cfg.Endpoint, path)

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to create request: %w", err)
	}

	signAndSetHeaders(req, http.MethodGet, path, cfg)

	log.Printf("[KMS-CLIENT] Wysyłanie żądania GET -> %s", url)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kms: request execution failed: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to read response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kms: unexpected status code %d fetching public key for %s, body: %s", res.StatusCode, targetService, string(bodyBytes))
	}

	var out publicKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to decode response JSON: %w", err)
	}

	if len(out.PublicKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("kms: invalid public key length: %d (expected %d)", len(out.PublicKeyBytes), ed25519.PublicKeySize)
	}

	return ed25519.PublicKey(out.PublicKeyBytes), nil
}

// #endregion

// #region Helpers & HMAC Signing
func signAndSetHeaders(req *http.Request, method, path string, cfg Config) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	payloadToSign := fmt.Sprintf("%s:%s:%s", method, path, timestamp)

	mac := hmac.New(sha256.New, []byte(cfg.ServiceSecret))
	mac.Write([]byte(payloadToSign))
	signatureHex := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set(HeaderServiceName, cfg.ServiceName)
	req.Header.Set(HeaderTimestamp, timestamp)
	req.Header.Set(HeaderHMACSignature, signatureHex)
	req.Header.Set(HeaderAccept, MIMEApplicationJSON)
	req.Header.Set(HeaderContentType, MIMEApplicationJSON)
}

// #endregion
