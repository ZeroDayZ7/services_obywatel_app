package config

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/zerodayz7/http-server/internal/shared/logger"
)

// RedisClient globalny klient Redis do odczytu JWT/sesji
var RedisClient *redis.Client

func NewRedisClient() *redis.Client {
	if RedisClient != nil {
		return RedisClient
	}

	log := logger.GetLogger()

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", AppConfig.Redis.Host, AppConfig.Redis.Port),
		Password: AppConfig.Redis.Password,
		DB:       AppConfig.Redis.DB,
	})

	// test połączenia
	if err := RedisClient.Ping(context.Background()).Err(); err != nil {
		log.Error("Failed to connect to Redis")
		panic(fmt.Sprintf("failed to connect to Redis: %v", err))
	}

	log.Info("Successfully connected to Redis")

	return RedisClient
}

// Helper do sprawdzania tokenów w Redis
func IsTokenValid(token string) bool {
	val, err := RedisClient.Get(context.Background(), token).Result()
	if err != nil {
		return false
	}
	return val == "valid"
}
