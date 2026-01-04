package redis

import (
	"context"
	"encoding/json"
	"time"
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

// SessionCache
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

// UpdateSessionFingerprint - poprawiona wersja z prefixami
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
