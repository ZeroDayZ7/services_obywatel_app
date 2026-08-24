package kms

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

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

	// Normalizacja ciała żądania
	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewBuffer(body)
	}

	req, err := http.NewRequestWithContext(reqCtx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("kms: failed to create request: %w", err)
	}

	req.Header.Set(HeaderAccept, MIMEApplicationJSON)
	if len(body) > 0 {
		req.Header.Set(HeaderContentType, MIMEApplicationJSON)
	}

	if sign {
		signAndSetHeaders(req, method, path, body, cfg)
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

func signAndSetHeaders(req *http.Request, method, path string, body []byte, cfg Config) {
	// 1. Obcinamy nanosekundy do pełnych sekund (eliminuje rozbieżności RFC3339 między Go a Rustem)
	timestamp := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	nonce := uuid.New().String()

	// 2. Zagwarantowanie pustego plasterka bajtów dla SHA-256 (GET / puste body)
	if len(body) == 0 {
		body = []byte{}
	}

	bodyHash := sha256.Sum256(body)
	bodyHashHex := hex.EncodeToString(bodyHash[:])

	// 3. Ścisły format nagłówka podpisu zgodny z Axum AuthenticatedService
	payloadToSign := fmt.Sprintf("%s:%s:%s:%s:%s", method, path, timestamp, nonce, bodyHashHex)

	mac := hmac.New(sha256.New, []byte(cfg.ServiceSecret))
	mac.Write([]byte(payloadToSign))
	signatureHex := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set(HeaderServiceName, cfg.ServiceName)
	req.Header.Set(HeaderTimestamp, timestamp)
	req.Header.Set(HeaderNonce, nonce)
	req.Header.Set(HeaderBodySHA256, bodyHashHex)
	req.Header.Set(HeaderHMACSignature, signatureHex)
}
