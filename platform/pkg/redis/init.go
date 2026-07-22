// cmdr: cmdr: redis\init.go

package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis(host, port, password string, db int) (*redis.Client, error) {
	if host == "" || port == "" {
		return nil, fmt.Errorf("redis host or port is empty (host: %s, port: %s)", host, port)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s:%s: %w", host, port, err)
	}

	RedisClient = client
	return client, nil
}

func IsTokenValid(token string) bool {
	val, err := RedisClient.Get(context.Background(), token).Result()
	if err != nil {
		return false
	}
	return val == "valid"
}
