package shared

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

// Pakiety danych dla poszczególnych usług
type PostgresCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RedisCredentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password"`
}

type MinioCredentials struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

// FullBootstrapResponse zawiera cały komplet danych na start
type FullBootstrapResponse struct {
	Postgres *PostgresCredentials `json:"postgres,omitempty"`
	Redis    *RedisCredentials    `json:"redis,omitempty"`
	Minio    *MinioCredentials    `json:"minio,omitempty"`
}

// BootstrapApp pobiera wszystkie wymagane sekrety na raz przy starcie usługi
func (c *Client) BootstrapApp(ctx context.Context, requiredServices []string) (*FullBootstrapResponse, error) {
	reqPayload, err := json.Marshal(map[string]any{
		"target_service": c.targetService,
		"services":       requiredServices,
	})
	if err != nil {
		return nil, fmt.Errorf("błąd serializacji żądania bootstrap: %w", err)
	}

	// 1. Wysyłamy komendę BOOTSTRAP z listą modułów
	cmd := fmt.Sprintf("BOOTSTRAP %s\n", string(reqPayload))
	rawResp, err := c.execCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("bootstrap nie powiódł się: %w", err)
	}

	var response FullBootstrapResponse
	if err := json.Unmarshal([]byte(rawResp), &response); err != nil {
		return nil, fmt.Errorf("błąd parsoowania bootstrap response: %w", err)
	}

	return &response, nil
}

// RefreshPostgres pobiera odświeżone dane TYLKO dla bazy danych w trakcie działania appki
func (c *Client) RefreshPostgres(ctx context.Context) (*PostgresCredentials, error) {
	cmd := fmt.Sprintf("REFRESH %s postgres\n", c.targetService)
	rawResp, err := c.execCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("rotacja poświadczeń DB nie powiodła się: %w", err)
	}

	var creds PostgresCredentials
	if err := json.Unmarshal([]byte(rawResp), &creds); err != nil {
		return nil, fmt.Errorf("błąd parsowania refreshed postgres creds: %w", err)
	}

	return &creds, nil
}

// Pomocnicze wykonanie dowolnej komendy UDS
func (c *Client) execCommand(ctx context.Context, command string) (string, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return "", fmt.Errorf("błąd połączenia z UDS: %w", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	if _, err := fmt.Fprint(conn, command); err != nil {
		return "", fmt.Errorf("błąd wysyłania komendy: %w", err)
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("błąd odczytu z UDS: %w", err)
	}

	response = strings.TrimSpace(response)
	if payload, ok := strings.CutPrefix(response, "OK "); ok {
		return payload, nil
	}

	return "", fmt.Errorf("błąd z agenta: %s", response)
}
