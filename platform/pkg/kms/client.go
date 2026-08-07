package kms

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
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

	PrivateKeyEndpoint   = "/api/v1/keys/private"
	PublicKeyEndpoint    = "/api/v1/keys/public/%s/%s"
	SymmetricKeyEndpoint = "/api/v1/keys/symmetric"

	HeaderServiceName   = "X-Service-Name"
	HeaderTimestamp     = "X-Timestamp"
	HeaderHMACSignature = "X-HMAC-Signature"
	HeaderAccept        = "Accept"
	HeaderContentType   = "Content-Type"

	MIMEApplicationJSON = "application/json"
	DefaultAlgorithm    = "Ed25519"
	AlgorithmAES256GCM  = "AES256GCM"
)

// ============================================================================
// TYPES
// ============================================================================

type Config struct {
	Endpoint      string        `json:"endpoint"`
	ServiceName   string        `json:"service_name"`
	ServiceSecret string        `json:"service_secret"`
	Timeout       time.Duration `json:"timeout"`
	HTTPClient    *http.Client  `json:"-"`
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
	ID           string `json:"id"`
	ServiceID    string `json:"service_id"`
	Algorithm    string `json:"algorithm"`
	Purpose      string `json:"purpose"`
	PublicKeyPEM string `json:"public_key_pem"`
	Version      int    `json:"version"`
	IsActive     bool   `json:"is_active"`
}

type getSymmetricKeyRequest struct {
	ServiceID string `json:"service_id"`
	Algorithm string `json:"algorithm"`
}

type symmetricKeyResponse struct {
	ServiceID string `json:"service_id"`
	Algorithm string `json:"algorithm"`
	Version   int    `json:"version"`
	KeyBytes  []byte `json:"key_bytes"`
}

var defaultHTTPClient = &http.Client{
	Timeout: DefaultTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

func getHTTPClient(cfg Config) *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return defaultHTTPClient
}

func executeRequest(ctx context.Context, cfg Config, method, path string, body []byte, sign bool) ([]byte, error) {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := fmt.Sprintf("%s%s", cfg.Endpoint, path)

	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewBuffer(body)
	}

	req, err := http.NewRequestWithContext(reqCtx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to create request: %w", err)
	}

	if sign {
		signAndSetHeaders(req, method, path, cfg)
	} else {
		req.Header.Set(HeaderAccept, MIMEApplicationJSON)
		if len(body) > 0 {
			req.Header.Set(HeaderContentType, MIMEApplicationJSON)
		}
	}

	res, err := getHTTPClient(cfg).Do(req)
	if err != nil {
		return nil, fmt.Errorf("kms: request execution failed: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to read response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kms: unexpected status code %d on %s %s, body: %s", res.StatusCode, method, path, string(bodyBytes))
	}

	return bodyBytes, nil
}

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

func FetchAuthPrivateKey(ctx context.Context, cfg Config, targetService string) (ed25519.PrivateKey, error) {
	if targetService == "" {
		targetService = cfg.ServiceName
	}

	reqBody, err := json.Marshal(getPrivateKeyRequest{
		ServiceID: targetService,
		Algorithm: DefaultAlgorithm,
	})
	if err != nil {
		return nil, fmt.Errorf("kms: failed to marshal request body: %w", err)
	}

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodPost, PrivateKeyEndpoint, reqBody, true)
	if err != nil {
		return nil, fmt.Errorf("kms: failed fetching private key for %s: %w", targetService, err)
	}

	var out privateKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to decode response JSON: %w", err)
	}

	if len(out.PrivateKeyBytes) == 0 {
		return nil, fmt.Errorf("kms: private_key_bytes is empty in KMS response payload")
	}

	var privKey ed25519.PrivateKey
	switch len(out.PrivateKeyBytes) {
	case ed25519.SeedSize:
		privKey = ed25519.NewKeyFromSeed(out.PrivateKeyBytes)
	case ed25519.PrivateKeySize:
		privKey = ed25519.PrivateKey(out.PrivateKeyBytes)
	default:
		return nil, fmt.Errorf("kms: invalid private key bytes length: %d (expected 32 or 64)", len(out.PrivateKeyBytes))
	}

	log.Printf("[KMS-CLIENT] ✅ Pobrano klucz PRYWATNY Ed25519 dla targetu '%s' (wersja %d)", out.ServiceID, out.Version)
	return privKey, nil
}

func FetchPublicKey(ctx context.Context, cfg Config, targetService string) (ed25519.PublicKey, error) {
	path := fmt.Sprintf(PublicKeyEndpoint, targetService, DefaultAlgorithm)

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodGet, path, nil, true)
	if err != nil {
		return nil, fmt.Errorf("kms: failed fetching public key for %s: %w", targetService, err)
	}

	var out publicKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to decode response JSON: %w", err)
	}

	if out.PublicKeyPEM == "" {
		return nil, fmt.Errorf("kms: public_key_pem is empty in KMS response payload")
	}

	block, _ := pem.Decode([]byte(out.PublicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("kms: failed to decode PEM block containing public key")
	}

	parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to parse PKIX public key: %w", err)
	}

	pubKey, ok := parsedKey.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("kms: parsed key is not of type ed25519.PublicKey (got %T)", parsedKey)
	}

	log.Printf("[KMS-CLIENT] ✅ Pomyślnie pobrano i przetworzono klucz PUBLICZNY PEM Ed25519 dla '%s' (wersja %d)", out.ServiceID, out.Version)
	return pubKey, nil
}

func FetchSymmetricKey(ctx context.Context, cfg Config, targetService string, algorithm string) ([]byte, error) {
	if targetService == "" {
		return nil, fmt.Errorf("kms: targetService cannot be empty for symmetric key request")
	}
	if algorithm == "" {
		algorithm = AlgorithmAES256GCM
	}

	reqBody, err := json.Marshal(getSymmetricKeyRequest{
		ServiceID: targetService,
		Algorithm: algorithm,
	})
	if err != nil {
		return nil, fmt.Errorf("kms: failed to marshal symmetric key request: %w", err)
	}

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodPost, SymmetricKeyEndpoint, reqBody, true)
	if err != nil {
		return nil, fmt.Errorf("kms: failed fetching symmetric key for %s (alg: %s): %w", targetService, algorithm, err)
	}

	var out symmetricKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to decode response JSON: %w", err)
	}

	if len(out.KeyBytes) == 0 {
		return nil, fmt.Errorf("kms: key_bytes is empty in KMS response payload")
	}

	log.Printf("[KMS-CLIENT] ✅ Pomyślnie pobrano klucz SYMETRYCZNY dla targetu '%s' [%s] (wersja %d)", out.ServiceID, out.Algorithm, out.Version)
	return out.KeyBytes, nil
}
