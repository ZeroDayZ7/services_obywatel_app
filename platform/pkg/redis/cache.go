// cmdr: redis/cache.go

package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("redis: key not found")

type Cache struct {
	client *Client
	ttl    time.Duration
}

//#region NewCache
func NewCache(client *Client, defaultTTL time.Duration) *Cache {
	return &Cache{
		client: client,
		ttl:    defaultTTL,
	}
}

//#region Set
func (c *Cache) Set(ctx context.Context, key string, value any, ttl ...time.Duration) error {
	d := c.ttl
	if len(ttl) > 0 {
		d = ttl[0]
	}
	return c.client.Set(ctx, key, value, d).Err()
}

//#region Get
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

//#region Del
func (c *Cache) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

//#region Exists
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}

// GetJSON pobiera wartość z klucza i deserializuje do wskazanego typu T
func GetJSON[T any](c *Cache, ctx context.Context, key string) (*T, error) {
	data, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var dest T
	if err := json.Unmarshal([]byte(data), &dest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal redis payload for key %s: %w", key, err)
	}

	return &dest, nil
}

// SetJSON serializuje podaną strukturę do JSON i zapisuje w Redisie
//#region SetJSON
func SetJSON(c *Cache, ctx context.Context, key string, value any, ttl ...time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal payload for redis key %s: %w", key, err)
	}
	return c.Set(ctx, key, data, ttl...)
}

//#region SendNotification
func (c *Cache) SendNotification(ctx context.Context, data any) error {
	return c.client.SendNotification(ctx, data)
}

// =========================================
// ================ AUDIT ==================
// =========================================

//#region ReadStream
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
		if errors.Is(err, goredis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res[0].Messages, nil
}

//#region AckStream
func (c *Client) AckStream(ctx context.Context, stream, group, messageID string) error {
	return c.XAck(ctx, stream, group, messageID).Err()
}

//#region SendNotification
func (c *Client) SendNotification(ctx context.Context, data any) error {
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

// =========================================
// ============ EVENT PUBLISHER =============
// =========================================

//#region Publish
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
