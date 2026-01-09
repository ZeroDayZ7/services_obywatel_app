package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ----------------------------
// CONFIG I CLIENT
// ----------------------------
type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type Client struct {
	*redis.Client
}

func New(cfg Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: 10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &Client{rdb}, nil
}

func (c *Client) Close() error {
	return c.Client.Close()
}

// ----------------------------
// STREAM BATCH METHODS
// ----------------------------

// ReadStreamBatch odczytuje maxCount elementów z grupy konsumentów w streamie
func (c *Client) ReadStreamBatch(
	ctx context.Context,
	stream, group, consumer string,
	maxCount int,
	block time.Duration,
) ([]redis.XMessage, error) {

	args := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Block:    block,
		Count:    int64(maxCount),
	}

	result, err := c.XReadGroup(ctx, args).Result()
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result[0].Messages, nil
}

// AckStreamBatch potwierdza batch wiadomości w streamie
func (c *Client) AckStreamBatch(
	ctx context.Context,
	stream, group string,
	ids []string,
) error {
	if len(ids) == 0 {
		return nil
	}
	return c.XAck(ctx, stream, group, ids...).Err()
}

func (c *Client) SendAuditLog(ctx context.Context, stream string, values map[string]any) error {
	_, err := c.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()
	return err
}

// Opcjonalnie: helper do tworzenia consumer group
func (c *Client) EnsureGroup(ctx context.Context, stream, group string) error {
	// ignorujemy błąd jeśli grupa już istnieje
	err := c.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}
