package kms

import (
	"context"
	"net/http"
)

func HealthCheck(ctx context.Context, cfg Config) error {
	_, err := executeRequest(ctx, cfg, http.MethodGet, "/health", nil, false)
	return err
}
