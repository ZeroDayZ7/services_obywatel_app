package di

import (
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/viper"
	"gorm.io/gorm"
)

type Container struct {
	Repos          *Repositories
	Services       *Services
	Handlers       *Handlers
	Redis          *redis.Client
	Cache          *redis.Cache
	EventPublisher rabbitmq.EventPublisher
	InternalSecret []byte
	Config         *viper.Config
}

func NewContainer(
	db *gorm.DB,
	redisClient *redis.Client,
	eventPublisher rabbitmq.EventPublisher,
	cfg *viper.Config,
) *Container {
	cache := redis.NewCache(
		redisClient,
		cfg.Session.TTL,
	)

	repos := NewRepositories(db)
	services := NewServices(repos, cache, eventPublisher, cfg)
	handlers := NewHandlers(services, cache, cfg)

	return &Container{
		Repos:          repos,
		Services:       services,
		Handlers:       handlers,
		Redis:          redisClient,
		Cache:          cache,
		EventPublisher: eventPublisher,
		InternalSecret: []byte(cfg.Internal.HMACSecret),
		Config:         cfg,
	}
}
