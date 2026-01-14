package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// TwoFASession przechowuje tymczasowe dane procesu weryfikacji dwuetapowej
type TwoFASession struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	CodeHash    string `json:"code_hash"`
	Token       string `json:"token"`
	Fingerprint string `json:"fingerprint"`
	Attempts    int    `json:"attempts"`
}

// --- Metody dla 2FA ---

// Set2FASession zapisuje sesję 2FA (kod i meta-dane) przed weryfikacją
func (c *Cache) Set2FASession(ctx context.Context, token string, sess TwoFASession, ttl time.Duration) error {
	data, _ := json.Marshal(sess)
	return c.client.Set(ctx, Login2FAPrefix+token, data, ttl).Err()
}

// Get2FASession pobiera sesję 2FA na podstawie tokenu
func (c *Cache) Get2FASession(ctx context.Context, token string) (*TwoFASession, error) {
	data, err := c.client.Get(ctx, Login2FAPrefix+token).Result()
	if err != nil {
		return nil, err
	}
	var sess TwoFASession
	if err := json.Unmarshal([]byte(data), &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// Delete2FASession usuwa sesję 2FA (np. po udanej weryfikacji lub po zablokowaniu)
func (c *Cache) Delete2FASession(ctx context.Context, token string) error {
	return c.client.Del(ctx, Login2FAPrefix+token).Err()
}

// Verify2FAAttempt zarządza licznikiem prób przy użyciu skryptu Lua.
// Zwraca status: "attempt_updated", "locked" lub "not_found".
func (c *Cache) Verify2FAAttempt(
	ctx context.Context,
	token string,
	maxAttempts int,
	ttl time.Duration,
) (string, error) {
	fullKey := Login2FAPrefix + token

	// Wykonujemy skrypt Lua, aby operacja inkrementacji i sprawdzenia limitu była atomowa
	res, err := c.client.Eval(
		ctx,
		verify2FAScript, // Skrypt musi być zdefiniowany w scripts.go
		[]string{fullKey},
		maxAttempts,
		int(ttl.Seconds()),
	).Result()
	if err != nil {
		return "", err
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) == 0 {
		return "", errors.New("invalid lua response from verify2fa script")
	}

	status := arr[0].(string)

	switch status {
	case "NOT_FOUND":
		return "not_found", nil
	case "LOCKED":
		return "locked", nil
	case "ATTEMPT_UPDATED":
		return "attempt_updated", nil
	default:
		return "", errors.New("unknown 2FA status from redis")
	}
}
