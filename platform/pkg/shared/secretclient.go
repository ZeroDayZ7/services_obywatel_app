package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Response struktura zwracana przez secret-agent
type SecretResponse struct {
	Status    string `json:"status"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	ExpiresAt string `json:"expires_at"`
}

type Client struct {
	httpClient *http.Client
	socketPath string
}

// NewClient tworzy klienta komunikującego się z sidecarem przez podany socket UDS
func NewClient(socketPath string) *Client {
	// Konfiguracja transportu HTTP przez Unix Domain Socket
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   5 * time.Second,
		},
		socketPath: socketPath,
	}
}

// GetCredentials pobiera poświadczenia do bazy z lokalnego sidecara
func (c *Client) GetCredentials(ctx context.Context) (*SecretResponse, error) {
	// Wysyłamy zapytanie HTTP pod dowolny adres (np. http://unix/credentials)
	// - transport i tak przekieruje je do pliku socketu.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/api/v1/credentials", nil)
	if err != nil {
		return nil, fmt.Errorf("błąd tworzenia żądania do agenta: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("błąd połączenia z agentem przez UDS (%s): %w", c.socketPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent zwrócił błąd HTTP status: %d", resp.StatusCode)
	}

	var secretResp SecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&secretResp); err != nil {
		return nil, fmt.Errorf("błąd parsowania odpowiedzi z agenta: %w", err)
	}

	return &secretResp, nil
}
