package redis

import (
	"context"
	"encoding/json"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Cache struct {
	client *Client
	ttl    time.Duration
}

func NewCache(client *Client, defaultTTL time.Duration) *Cache {
	return &Cache{
		client: client,
		ttl:    defaultTTL,
	}
}

func (c *Cache) Set(ctx context.Context, key string, value any, ttl ...time.Duration) error {
	d := c.ttl
	if len(ttl) > 0 {
		d = ttl[0]
	}
	// ZMIANA: Usunięto c.key()
	return c.client.Set(ctx, key, value, d).Err()
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	// ZMIANA: Usunięto c.key()
	return c.client.Get(ctx, key).Result()
}

func (c *Cache) Del(ctx context.Context, key string) error {
	// ZMIANA: Usunięto c.key()
	return c.client.Del(ctx, key).Err()
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	// ZMIANA: Usunięto c.key()
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}

func (c *Cache) SendNotification(ctx context.Context, data any) error {
	// Cache używa swojego wewnętrznego klienta, aby wysłać powiadomienie
	return c.client.SendNotification(ctx, data)
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
