package kms

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Config struct {
	Endpoint      string        `json:"endpoint"`       // np. "http://kms-service:8080"
	ServiceName   string        `json:"service_name"`   // np. "api-gateway"
	ServiceSecret string        `json:"service_secret"` // token/secret Gatewaya do KMS
	Timeout       time.Duration `json:"timeout"`
}

type keyResponse struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key,omitempty"`
}

// FetchAuthPrivateKey pobiera klucz prywatny (dla auth-service)
func FetchAuthPrivateKey(ctx context.Context, cfg Config) (ed25519.PrivateKey, error) {
	resp, err := requestKey(ctx, cfg, cfg.ServiceName, "private")
	if err != nil {
		return nil, fmt.Errorf("kms: failed to fetch private key: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(resp.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("kms: invalid base64 private key: %w", err)
	}

	if len(data) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("kms: invalid private key length: %d", len(data))
	}

	return ed25519.PrivateKey(data), nil
}

// FetchPublicKey pobiera klucz publiczny docelowego serwisu (np. auth-service)
func FetchPublicKey(ctx context.Context, cfg Config, targetService string) (ed25519.PublicKey, error) {
	resp, err := requestKey(ctx, cfg, targetService, "public")
	if err != nil {
		return nil, fmt.Errorf("kms: failed to fetch public key for %s: %w", targetService, err)
	}

	data, err := base64.StdEncoding.DecodeString(resp.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("kms: invalid base64 public key: %w", err)
	}

	if len(data) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("kms: invalid public key length: %d", len(data))
	}

	return ed25519.PublicKey(data), nil
}

func requestKey(ctx context.Context, cfg Config, targetService string, keyType string) (*keyResponse, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}

	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	url := fmt.Sprintf("%s/v1/keys/%s?type=%s", cfg.Endpoint, targetService, keyType)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Nagłówki legitymujące serwis pobierający (np. api-gateway) przed KMS
	req.Header.Set("X-Service-Name", cfg.ServiceName)
	req.Header.Set("X-Service-Secret", cfg.ServiceSecret)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	var out keyResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}
