package di

import "github.com/zerodayz7/platform/pkg/redis"

type Container struct {
	Redis *redis.Client
	Cache *redis.Cache
}

func NewContainer(redisClient *redis.Client, cache *redis.Cache) *Container {
	return &Container{
		Redis: redisClient,
		Cache: cache,
	}
}
