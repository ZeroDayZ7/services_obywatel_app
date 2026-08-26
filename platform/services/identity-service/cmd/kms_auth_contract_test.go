package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zerodayz7/platform/pkg/kms"
)

//#region TestIdentityKMSRequestContractMatchesRustBackend
func TestIdentityKMSRequestContractMatchesRustBackend(t *testing.T) {
	ctx := context.Background()
	cfg := kms.Config{
		Endpoint:      "http://127.0.0.1:8081",
		ServiceName:   "identity-service",
		ServiceSecret: "super-long-random-secret-for-identity-service-hmac-64-bytes",
		Timeout:       2 * time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method mismatch: got %q want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/api/v1/keys/symmetric" {
			t.Fatalf("path mismatch: got %q want %q", r.URL.Path, "/api/v1/keys/symmetric")
		}
		if got := r.Header.Get(kms.HeaderServiceName); got != cfg.ServiceName {
			t.Fatalf("X-Service-Name mismatch: got %q want %q", got, cfg.ServiceName)
		}
		if got := r.Header.Get(kms.HeaderContentType); got != kms.MIMEApplicationJSON {
			t.Fatalf("content-type mismatch: got %q want %q", got, kms.MIMEApplicationJSON)
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var requestBody kms.SymmetricKeyRequest
		if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if requestBody.ServiceID != "hmac-gateway-identity" {
			t.Fatalf("service_id mismatch: got %q want %q", requestBody.ServiceID, "hmac-gateway-identity")
		}
		if requestBody.Algorithm != "HmacSha256" {
			t.Fatalf("algorithm mismatch: got %q want %q", requestBody.Algorithm, "HmacSha256")
		}

		timestamp := r.Header.Get(kms.HeaderTimestamp)
		nonce := r.Header.Get(kms.HeaderNonce)
		bodyHashHex := r.Header.Get(kms.HeaderBodySHA256)
		if timestamp == "" || nonce == "" || bodyHashHex == "" {
			t.Fatalf("missing required signing headers: timestamp=%q nonce=%q bodyHash=%q", timestamp, nonce, bodyHashHex)
		}
		if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
			t.Fatalf("invalid timestamp format: %q: %v", timestamp, err)
		}
		sum := sha256.Sum256(bodyBytes)
		if bodyHashHex != hex.EncodeToString(sum[:]) {
			t.Fatalf("X-Body-SHA256 mismatch: got %q want %q", bodyHashHex, hex.EncodeToString(sum[:]))
		}

		payloadToSign := fmt.Sprintf("%s:%s:%s:%s:%s", r.Method, r.URL.Path, timestamp, nonce, bodyHashHex)
		expected := hmac.New(sha256.New, []byte(cfg.ServiceSecret))
		_, _ = expected.Write([]byte(payloadToSign))
		if got := hex.EncodeToString(expected.Sum(nil)); got != r.Header.Get(kms.HeaderHMACSignature) {
			t.Fatalf("signature mismatch: got %q want %q", r.Header.Get(kms.HeaderHMACSignature), got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service_id":"hmac-gateway-identity","algorithm":"HmacSha256","version":1,"key_b64":"a2V5"}`))
	}))
	defer server.Close()

	cfg.Endpoint = server.URL
	keyBytes, version, err := kms.FetchSymmetricKeyWithVersion(ctx, cfg, "hmac-gateway-identity", 1, "HmacSha256")
	if err != nil {
		t.Fatalf("FetchSymmetricKeyWithVersion returned error: %v", err)
	}
	if version != 1 {
		t.Fatalf("version mismatch: got %d want %d", version, 1)
	}
	if string(keyBytes) != "key" {
		t.Fatalf("decoded key mismatch: got %q want %q", string(keyBytes), "key")
	}
}
