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

// #region FetchAuthPrivateKey
func FetchAuthPrivateKey(ctx context.Context, cfg Config, targetService string) (ed25519.PrivateKey, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	if targetService == "" {
		targetService = cfg.ServiceName
	}

	reqBody := getPrivateKeyRequest{
		ServiceID: targetService,
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
		return nil, fmt.Errorf("kms: unexpected status code %d fetching private key for %s, body: %s", res.StatusCode, targetService, string(bodyBytes))
	}

	var out privateKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to decode response JSON: %w", err)
	}

	if len(out.PrivateKeyBytes) == 0 {
		return nil, fmt.Errorf("kms: private_key_bytes is empty in KMS response payload")
	}

	var privKey ed25519.PrivateKey
	if len(out.PrivateKeyBytes) == ed25519.SeedSize {
		privKey = ed25519.NewKeyFromSeed(out.PrivateKeyBytes)
	} else if len(out.PrivateKeyBytes) == ed25519.PrivateKeySize {
		privKey = ed25519.PrivateKey(out.PrivateKeyBytes)
	} else {
		return nil, fmt.Errorf("kms: invalid private key bytes length: %d (expected 32 or 64)", len(out.PrivateKeyBytes))
	}

	log.Printf("[KMS-CLIENT] ✅ Pobrano klucz PRYWATNY Ed25519 dla targetu '%s' (wersja %d)", out.ServiceID, out.Version)
	return privKey, nil
}

// #endregion

// #region FetchPublicKey
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

// #endregion

// #region FetchSymmetricKey
// FetchSymmetricKey pobiera klucz symetryczny HMAC/AES z KMS przy starcie serwisu.
// Przyjmuje algorytm (np. "AES256GCM" lub "HmacSha256"). Jeśli przekazano "", domyślnie używa AES256GCM.
func FetchSymmetricKey(ctx context.Context, cfg Config, targetService string, algorithm string) ([]byte, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	if targetService == "" {
		return nil, fmt.Errorf("kms: targetService cannot be empty for symmetric key request")
	}

	if algorithm == "" {
		algorithm = AlgorithmAES256GCM
	}

	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	reqBody := getSymmetricKeyRequest{
		ServiceID: targetService,
		Algorithm: algorithm,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to marshal symmetric key request: %w", err)
	}

	path := SymmetricKeyEndpoint
	url := fmt.Sprintf("%s%s", cfg.Endpoint, path)

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("kms: failed to create request: %w", err)
	}

	// Generujemy i ustawiamy nagłówki autoryzacyjne
	signAndSetHeaders(req, http.MethodPost, path, cfg)

	// LOGI DIAGNOSTYCZNE PRZED WYSŁANIEM ŻĄDANIA
	log.Printf("[KMS-CLIENT] 🔍 --- ODBICIE ŻĄDANIA KMS ---")
	log.Printf("[KMS-CLIENT] 🌐 URL: %s %s", req.Method, url)
	log.Printf("[KMS-CLIENT] 📦 Payload JSON: %s", string(jsonBytes))
	log.Printf("[KMS-CLIENT] 🔑 X-Service-Name: %s", req.Header.Get(HeaderServiceName))
	log.Printf("[KMS-CLIENT] ⏰ X-Timestamp: %s", req.Header.Get(HeaderTimestamp))
	log.Printf("[KMS-CLIENT] ✍️  X-HMAC-Signature: %s", req.Header.Get(HeaderHMACSignature))
	log.Printf("[KMS-CLIENT] ---------------------------------")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("[KMS-CLIENT] ❌ Błąd sieciowy/wykonania requestu: %v", err)
		return nil, fmt.Errorf("kms: request execution failed: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to read response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		log.Printf("[KMS-CLIENT] ❌ ODPOWIEDŹ BŁĘDU KMS [%d]", res.StatusCode)
		log.Printf("[KMS-CLIENT] 📩 Response Body: %s", string(bodyBytes))
		log.Printf("[KMS-CLIENT] ---------------------------------")
		return nil, fmt.Errorf("kms: unexpected status code %d fetching symmetric key for %s (alg: %s), body: %s", res.StatusCode, targetService, algorithm, string(bodyBytes))
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
