package kms

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zerodayz7/platform/pkg/shared"
)

const (
	SignEndpoint = "/api/v1/keys/sign"
)

type signDataRequest struct {
	TargetService string  `json:"target_service"`
	Algorithm     string  `json:"algorithm"`
	PayloadB64    string  `json:"payload_b64"`
	KeyVersion    *uint32 `json:"key_version,omitempty"`
}

type signDataResponse struct {
	SignatureB64 string `json:"signature_b64"`
	KeyVersion   uint32 `json:"key_version"`
	Algorithm    string `json:"algorithm"`
}

func SignData(ctx context.Context, cfg Config, targetService string, algorithm string, payload []byte) ([]byte, uint32, error) {
	if targetService == "" {
		targetService = cfg.ServiceName
	}
	if algorithm == "" {
		algorithm = DefaultAlgorithm
	}

	payloadB64 := base64.StdEncoding.EncodeToString(payload)

	reqBody, err := json.Marshal(signDataRequest{
		TargetService: targetService,
		Algorithm:     algorithm,
		PayloadB64:    payloadB64,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("kms: failed to marshal sign request: %w", err)
	}

	bodyBytes, err := executeRequest(ctx, cfg, http.MethodPost, SignEndpoint, reqBody, true)
	if err != nil {
		return nil, 0, fmt.Errorf("kms: remote signing request failed: %w", err)
	}

	var out signDataResponse
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, 0, fmt.Errorf("kms: failed to decode sign response JSON: %w", err)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(out.SignatureB64)
	if err != nil {
		return nil, 0, fmt.Errorf("kms: failed to decode signature base64: %w", err)
	}

	shared.GetLogger().Info(fmt.Sprintf("[KMS-CLIENT] ✅ Zdalnie podpisano dane kluczem '%s' [%s] (wersja %d)", targetService, out.Algorithm, out.KeyVersion))
	return sigBytes, out.KeyVersion, nil
}
