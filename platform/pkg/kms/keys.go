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

// FetchPublicKey pobiera klucz publiczny.
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

// FetchSymmetricKey pobiera klucz symetryczny dla wskazanego serwisu/celu.
func FetchSymmetricKey(ctx context.Context, cfg Config, serviceID string, algorithm ...string) ([]byte, error) {
	rawKey, _, err := fetchSymmetricKeyInternal(ctx, cfg, serviceID, algorithm...)
	return rawKey, err
}

// FetchSymmetricKeyWithVersion zachowuje wsteczną kompatybilność z wywołaniami przesyłającymi wersję (np. version=1).
func FetchSymmetricKeyWithVersion(ctx context.Context, cfg Config, serviceID string, version int, algorithm ...string) ([]byte, int, error) {
	return fetchSymmetricKeyInternal(ctx, cfg, serviceID, algorithm...)
}

func fetchSymmetricKeyInternal(ctx context.Context, cfg Config, serviceID string, algorithm ...string) ([]byte, int, error) {
	algo := "AES256GCM"
	if len(algorithm) > 0 && algorithm[0] != "" {
		algo = algorithm[0]
	}

	reqPayload := SymmetricKeyRequest{
		ServiceID: serviceID,
		Algorithm: algo,
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, 0, fmt.Errorf("kms: failed to marshal symmetric key request: %w", err)
	}

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodPost, EndpointSymmetric, reqBody, true)
	if err != nil {
		return nil, 0, fmt.Errorf("kms: failed fetching symmetric key for service %s: %w", serviceID, err)
	}

	var out SymmetricKeyResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, 0, fmt.Errorf("kms: failed to decode symmetric key response JSON: %w", err)
	}

	rawKey, err := base64.StdEncoding.DecodeString(out.KeyB64)
	if err != nil {
		return nil, 0, fmt.Errorf("kms: failed to decode base64 symmetric key: %w", err)
	}

	return rawKey, out.Version, nil
}
