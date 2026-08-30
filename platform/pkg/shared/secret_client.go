package shared

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// SecretResponse reprezentuje pobrane dane uwierzytelniające
type SecretResponse struct {
	Username string
	Password string
}

// Client obsługuje surową komunikację tekstową z secret-agent przez UDS
type Client struct {
	socketPath    string
	targetService string // np. "postgres_auth"
}

// NewClient tworzy instancję klienta UDS
func NewClient(socketPath string, targetService string) *Client {
	if targetService == "" {
		targetService = "postgres_auth" // Domyślny prefiks kluczy w cache
	}
	return &Client{
		socketPath:    socketPath,
		targetService: targetService,
	}
}

// getSecret wysyła zapytanie o pojedynczy klucz do agenta
func (c *Client) getSecret(ctx context.Context, key string) (string, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return "", fmt.Errorf("błąd połączenia z gniazdem UDS (%s): %w", c.socketPath, err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	}

	// 1. Wysyłamy klucz: <KEY>\n
	if _, err := fmt.Fprintf(conn, "%s\n", key); err != nil {
		return "", fmt.Errorf("błąd zapisu klucza [%s] do socketu: %w", key, err)
	}

	// 2. Odczytujemy odpowiedź: OK <VALUE>\n lub ERR <REASON>\n
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("błąd odczytu odpowiedzi dla klucza [%s]: %w", key, err)
	}

	response = strings.TrimSpace(response)

	if after, ok := strings.CutPrefix(response, "OK "); ok {
		return after, nil
	}

	return "", fmt.Errorf("secret-agent zwrócił błąd dla [%s]: %s", key, response)
}

// GetCredentials pobiera login i hasło dla skonfigurowanej usługi
func (c *Client) GetCredentials(ctx context.Context) (*SecretResponse, error) {
	userKey := fmt.Sprintf("%s_username", c.targetService)
	passKey := fmt.Sprintf("%s_password", c.targetService)

	username, err := c.getSecret(ctx, userKey)
	if err != nil {
		return nil, fmt.Errorf("pobieranie username nie powiodło się: %w", err)
	}

	password, err := c.getSecret(ctx, passKey)
	if err != nil {
		return nil, fmt.Errorf("pobieranie password nie powiodło się: %w", err)
	}

	return &SecretResponse{
		Username: username,
		Password: password,
	}, nil
}
