package redis

import (
	"context"
	"time"
)

type Cache struct {
	client *Client
	prefix string
	ttl    time.Duration
}

func NewCache(client *Client, prefix string, defaultTTL time.Duration) *Cache {
	return &Cache{client: client, prefix: prefix, ttl: defaultTTL}
}
func (c *Cache) key(k string) string { return c.prefix + k }

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
