package di

import (
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"gorm.io/gorm"
)

type Container struct {
	Repos    *Repositories
	Services *Services
	Handlers *Handlers
	Redis    *redis.Client
	Cache    *redis.Cache
}

func NewContainer(db *gorm.DB, redisClient *redis.Client) *Container {
	repos := NewRepositories(db)
	services := NewServices(repos)

	cache := redis.NewCache(
		redisClient,
		redis.SessionPrefix,
		config.AppConfig.SessionTTL,
	)

	handlers := NewHandlers(services, cache)

	return &Container{
		Repos:    repos,
		Services: services,
		Handlers: handlers,
		Redis:    redisClient,
		Cache:    cache,
	}
}
