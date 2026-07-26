// cmdr: redis/client.go

package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	fiberRedis "github.com/gofiber/storage/redis/v3"
	goredis "github.com/redis/go-redis/v9"
)

type Config struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	PoolTimeout  time.Duration
	Timeout      time.Duration
}

type Client struct {
	*goredis.Client
}

func New(cfg Config) (*Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		PoolTimeout:  cfg.PoolTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &Client{rdb}, nil
}

func (c *Client) Close() error {
	return c.Client.Close()
}

func (c *Client) AsFiberStorage() fiber.Storage {
	return fiberRedis.NewFromConnection(c.Client)
}

// ----------------------------
// STREAM BATCH METHODS
// ----------------------------

func (c *Client) ReadStreamBatch(
	ctx context.Context,
	stream, group, consumer string,
	maxCount int,
	block time.Duration,
) ([]goredis.XMessage, error) {
	args := &goredis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Block:    block,
		Count:    int64(maxCount),
	}

	result, err := c.XReadGroup(ctx, args).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result[0].Messages, nil
}

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
	_, err := c.XAdd(ctx, &goredis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()
	return err
}

func (c *Client) EnsureGroup(ctx context.Context, stream, group string) error {
	err := c.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}
