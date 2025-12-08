package di

import (
	"github.com/redis/go-redis/v9"
)

// Container przechowuje wszystkie zależności mikroserwisu
type Container struct {
	RedisClient *redis.Client
	// inne serwisy np. UserService, Repozytoria...
}

// NewContainer tworzy kontener
func NewContainer(redisClient *redis.Client) *Container {
	return &Container{
		RedisClient: redisClient,
	}
}
