package di

import (
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/config"
	"gorm.io/gorm"
)

type Container struct {
	Handlers *Handlers
	Workers  *Workers
	Services *Services
	Consumers *Consumers
	Redis    *redis.Client
	Logger   *shared.Logger
	KeyStore *httpserver.KeyStore
	Config   *config.Config
	// Optional RabbitMQ publisher / event plumbing will be added during bootstrap
	EventPublisher interface{}
}

func NewContainer(db *gorm.DB, redisClient *redis.Client, log *shared.Logger, cfg *config.Config, keyStore *httpserver.KeyStore) *Container {
	repos := NewRepositories(db)
	services := NewServices(repos)

	handlers := NewHandlers(services)
	workers := NewWorkers(redisClient, services, log)
	consumers := NewConsumers(services)

	return &Container{
		Handlers: handlers,
		Workers:  workers,
		Services: services,
		Consumers: consumers,
		Redis:    redisClient,
		Logger:   log,
		KeyStore: keyStore,
		Config:   cfg,
		EventPublisher: nil,
	}
}