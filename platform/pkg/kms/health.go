// cmdr: kms\health.go

package kms

import (
	"context"
	"net/http"
)

// #region HealthCheck
//#region HealthCheck
func HealthCheck(ctx context.Context, cfg Config) error {
	_, err := executeRequest(ctx, cfg, http.MethodGet, "/health", nil, false)
	return err
}
