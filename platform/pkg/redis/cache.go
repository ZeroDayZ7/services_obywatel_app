package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Cache struct {
	client *Client
	prefix string
	ttl    time.Duration
}

func NewCache(client *Client, prefix string, defaultTTL time.Duration) *Cache {
	return &Cache{
		client: client,
		prefix: prefix,
		ttl:    defaultTTL,
	}
}

type UserSession struct {
	UserID      string `json:"user_id"`
	Fingerprint string `json:"fingerprint"`
}

type TwoFASession struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	CodeHash    string `json:"code_hash"`
	Token       string `json:"token"`
	Fingerprint string `json:"fingerprint"`
	Attempts    int    `json:"attempts"`
}

func (c *Cache) key(k string) string {
	return c.prefix + k
}

// SetSession teraz przyjmuje fingerprint i zapisuje JSON
func (c *Cache) SetSession(ctx context.Context, sessionID string, userID string, fingerprint string, ttl time.Duration) error {
	session := UserSession{
		UserID:      userID,
		Fingerprint: fingerprint,
	}

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	// Używamy c.Set, która już dodaje prefix
	return c.Set(ctx, sessionID, data, ttl)
}

func (c *Cache) GetUserIDBySession(ctx context.Context, sessionID string) (string, error) {
	return c.Get(ctx, sessionID)
}

func (c *Cache) DeleteSession(ctx context.Context, sessionID string) error {
	return c.Del(ctx, sessionID)
}

func (c *Cache) Set(ctx context.Context, key string, value any, ttl ...time.Duration) error {
	d := c.ttl
	if len(ttl) > 0 {
		d = ttl[0]
	}
	return c.client.Set(ctx, c.key(key), value, d).Err()
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, c.key(key)).Result()
}

func (c *Cache) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.key(key)).Err()
}

func (c *Cache) TTL() time.Duration {
	return c.ttl
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, c.key(key)).Result()
	return n > 0, err
}

func (c *Cache) SendNotification(ctx context.Context, data any) error {
	// Cache używa swojego wewnętrznego klienta, aby wysłać powiadomienie
	return c.client.SendNotification(ctx, data)
}

// UpdateSessionFingerpriEnsureGroupnt - poprawiona wersja z prefixami
func (c *Cache) UpdateSessionFingerprint(ctx context.Context, sessionID string, newFingerprint string) error {
	// 1. Pobieramy dane używając c.Get (obsługuje prefixy)
	data, err := c.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	var session UserSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return err
	}

	// 2. Aktualizacja
	session.Fingerprint = newFingerprint

	// 3. Pobranie TTL dla klucza z prefixem
	fullKey := c.key(sessionID)
	ttl, _ := c.client.TTL(ctx, fullKey).Result()

	// 4. Zapis z powrotem przez c.Set
	updatedData, _ := json.Marshal(session)
	return c.Set(ctx, sessionID, updatedData, ttl)
}

func (c *Cache) GetSession(ctx context.Context, sessionID string) (*UserSession, error) {
	data, err := c.Get(ctx, sessionID) // c.Get automatycznie dodaje prefix
	if err != nil {
		return nil, err
	}

	var session UserSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// =========================================
// ================  AUDIT  ================
// =========================================
func (c *Client) ReadStream(
	ctx context.Context,
	stream string,
	group string,
	consumer string,
) ([]goredis.XMessage, error) {
	res, err := c.XReadGroup(ctx, &goredis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    10,
		Block:    5 * time.Second,
	}).Result()
	if err != nil {
		if err == goredis.Nil {
			return nil, nil
		}
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res[0].Messages, nil
}

// AckStream potwierdza przetworzenie wiadomości (XAck)
func (c *Client) AckStream(ctx context.Context, stream, group, messageID string) error {
	return c.XAck(ctx, stream, group, messageID).Err()
}

// SendNotification wysyła dane do notification_stream
func (c *Client) SendNotification(ctx context.Context, data any) error {
	// Pomijamy bootstrapowe eventy (jeśli potrzebne)
	if m, ok := data.(map[string]any); ok {
		if b, exists := m["_bootstrap"]; exists {
			if isBootstrap, ok := b.(bool); ok && isBootstrap {
				return nil
			}
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return c.XAdd(ctx, &goredis.XAddArgs{
		Stream: "notification_stream",
		Values: map[string]any{
			"payload": string(jsonData),
		},
	}).Err()
}

// #region EVENT PUBLISHER

// =========================================
// ============ EVENT PUBLISHER =============
// =========================================

// Publish implements events.StreamPublisher
func (c *Cache) Publish(ctx context.Context, stream string, payload any) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return c.client.XAdd(ctx, &goredis.XAddArgs{
		Stream: stream,
		Values: map[string]any{
			"payload": string(jsonData),
		},
	}).Err()
}

func (c *Cache) Verify2FAAttempt(
	ctx context.Context,
	key string,
	maxAttempts int,
	ttl time.Duration,
) (string, error) {
	res, err := c.client.Eval(
		ctx,
		verify2FAScript,
		[]string{key},
		maxAttempts,
		int(ttl.Seconds()),
	).Result()
	if err != nil {
		return "", err
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) == 0 {
		return "", errors.New("invalid lua response")
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
		return "", errors.New("unknown lua status")
	}
}
