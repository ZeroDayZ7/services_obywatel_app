package kms

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
)

// CheckHealth sprawdza status dostępności serwisu KMS (bez podpisu HMAC).
func CheckHealth(ctx context.Context, cfg Config) error {
	_, err := executeRequest(ctx, cfg, http.MethodGet, EndpointHealth, nil, false)
	return err
}

// FetchPublicKey pobiera klucz publiczny. Obsługuje warianty wywołania z 3 lub 4 parametrami poprzez wartosci domyslne.
func FetchPublicKey(ctx context.Context, cfg Config, targetService string, algorithm ...string) (ed25519.PublicKey, error) {
	if targetService == "" {
		targetService = cfg.ServiceName
	}

	algo := DefaultAlgorithm
	if len(algorithm) > 0 && algorithm[0] != "" {
		algo = algorithm[0]
	}

	path := fmt.Sprintf(EndpointPublic, targetService, algo)

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodGet, path, nil, true)
	if err != nil {
		return nil, fmt.Errorf("kms: failed fetching public key for %s: %w", targetService, err)
	}

	var out KeyResponse
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

	return pubKey, nil
}

// FetchAuthPrivateKey pobiera klucz prywatny Ed25519 do podpisywania tokenów/materiału autoryzacyjnego.
func FetchAuthPrivateKey(ctx context.Context, cfg Config) (ed25519.PrivateKey, error) {
	bodyBytes, err := executeRequest(ctx, cfg, http.MethodGet, EndpointAuthPrivate, nil, true)
	if err != nil {
		return nil, fmt.Errorf("kms: failed fetching auth private key: %w", err)
	}

	var out PrivateKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to decode private key response JSON: %w", err)
	}

	block, _ := pem.Decode([]byte(out.PrivateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("kms: failed to decode PEM block containing private key")
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to parse PKCS8 private key: %w", err)
	}

	privKey, ok := parsedKey.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("kms: parsed key is not of type ed25519.PrivateKey (got %T)", parsedKey)
	}

	return privKey, nil
}

// FetchSymmetricKey pobiera najnowszą wersję klucza symetrycznego dla danego celu (purpose).
func FetchSymmetricKey(ctx context.Context, cfg Config, purpose string) ([]byte, error) {
	path := fmt.Sprintf(EndpointSymmetric, purpose)

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodGet, path, nil, true)
	if err != nil {
		return nil, fmt.Errorf("kms: failed fetching symmetric key for purpose %s: %w", purpose, err)
	}

	var out SymmetricKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, fmt.Errorf("kms: failed to decode symmetric key response JSON: %w", err)
	}

	rawKey, err := base64.StdEncoding.DecodeString(out.KeyBase64)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to decode base64 symmetric key: %w", err)
	}

	return rawKey, nil
}

// FetchSymmetricKeyWithVersion pobiera konkretną wersję klucza symetrycznego. Zwraca bajty klucza oraz wersję.
func FetchSymmetricKeyWithVersion(ctx context.Context, cfg Config, purpose string, version int) ([]byte, int, error) {
	path := fmt.Sprintf(EndpointSymmetricWithVersion, purpose, version)

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodGet, path, nil, true)
	if err != nil {
		return nil, 0, fmt.Errorf("kms: failed fetching symmetric key for purpose %s version %d: %w", purpose, version, err)
	}

	var out SymmetricKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, 0, fmt.Errorf("kms: failed to decode symmetric key response JSON: %w", err)
	}

	rawKey, err := base64.StdEncoding.DecodeString(out.KeyBase64)
	if err != nil {
		return nil, 0, fmt.Errorf("kms: failed to decode base64 symmetric key: %w", err)
	}

	return rawKey, out.Version, nil
}
