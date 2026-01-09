package di

import (
	"time"

	"github.com/zerodayz7/platform/pkg/redis"
	"gorm.io/gorm"
)

type Container struct {
	Repos    *Repositories
	Services *Services
	Handlers *Handlers
	Redis    *redis.Client
	Cache    *redis.Cache
}

func NewContainer(
	db *gorm.DB,
	redisClient *redis.Client,
	sessionPrefix string,
	sessionTTL time.Duration,
) *Container {
	repos := NewRepositories(db)
	services := NewServices(repos)

	cache := redis.NewCache(
		redisClient,
		sessionPrefix,
		sessionTTL,
	)

	handlers := NewHandlers(services, cache, sessionTTL)

	return &Container{
		Repos:    repos,
		Services: services,
		Handlers: handlers,
		Redis:    redisClient,
		Cache:    cache,
	}
}
