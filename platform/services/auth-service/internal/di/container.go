package di

import (
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"gorm.io/gorm"
)

type Container struct {
	Repos          *Repositories
	Services       *Services
	Handlers       *Handlers
	Consumers      *Consumers
	Redis          *redis.Client
	Cache          *redis.Cache
	EventPublisher rabbitmq.EventPublisher
	KeyStore       *httpserver.KeyStore
	Config         *config.Config
}

func NewContainer(
	db *gorm.DB,
	redisClient *redis.Client,
	eventPublisher rabbitmq.EventPublisher,
	cfg *config.Config,
	keyStore *httpserver.KeyStore,
) *Container {
	cache := redis.NewCache(
		redisClient,
		cfg.Session.TTL,
	)

	repos := NewRepositories(db)
	services := NewServices(repos, cache, cfg)
	handlers := NewHandlers(services, cache, cfg)
	consumers := NewConsumers(services)

	return &Container{
		Repos:          repos,
		Services:       services,
		Handlers:       handlers,
		Consumers:      consumers,
		Redis:          redisClient,
		Cache:          cache,
		EventPublisher: eventPublisher,
		KeyStore:       keyStore,
		Config:         cfg,
	}
}
