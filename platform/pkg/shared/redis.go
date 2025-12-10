package shared

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis(host, port, password string, db int) *redis.Client {
	if RedisClient != nil {
		return RedisClient
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})

	if err := RedisClient.Ping(context.Background()).Err(); err != nil {
		panic(fmt.Sprintf("failed to connect to Redis: %v", err))
	}

	return RedisClient
}

func IsTokenValid(token string) bool {
	val, err := RedisClient.Get(context.Background(), token).Result()
	if err != nil {
		return false
	}
	return val == "valid"
}
