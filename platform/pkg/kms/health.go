package kms

import (
	"context"
	"fmt"
	"net/http"
)

// #region HealthCheck
// HealthCheck wysyła proste żądanie do KMS sprawdzające, czy serwis działa i odpowiada.
func HealthCheck(ctx context.Context, cfg Config) error {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	url := fmt.Sprintf("%s/health", cfg.Endpoint)

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("kms: failed to create healthcheck request: %w", err)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("kms service unreachable at %s: %w", cfg.Endpoint, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("kms healthcheck returned unexpected status: %d", res.StatusCode)
	}

	return nil
}

// #endregion
