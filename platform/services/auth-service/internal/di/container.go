package di

import (
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/types"
	"gorm.io/gorm"
)

type Container struct {
	Repos          *Repositories
	Services       *Services
	Handlers       *Handlers
	Redis          *redis.Client
	Cache          *redis.Cache
	InternalSecret []byte
	Config         *types.Config
}

func NewContainer(db *gorm.DB, redisClient *redis.Client, cfg *types.Config) *Container {
	repos := NewRepositories(db)
	services := NewServices(repos, cfg)

	cache := redis.NewCache(
		redisClient,
		cfg.Session.Prefix,
		cfg.Session.TTL,
	)

	handlers := NewHandlers(services, cache, cfg)

	return &Container{
		Repos:          repos,
		Services:       services,
		Handlers:       handlers,
		Redis:          redisClient,
		Cache:          cache,
		InternalSecret: []byte(cfg.Internal.HMACSecret),
		Config:         cfg,
	}
}
