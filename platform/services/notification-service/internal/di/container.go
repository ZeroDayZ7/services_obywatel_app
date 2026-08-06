package di

import (
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/config"
	"gorm.io/gorm"
)

type Container struct {
	Handlers *Handlers
	Workers  *Workers
	Redis    *redis.Client
	Logger   *shared.Logger
	Config   *config.Config
}

func NewContainer(db *gorm.DB, redisClient *redis.Client, log *shared.Logger, cfg *config.Config) *Container {
	repos := NewRepositories(db)
	services := NewServices(repos)

	handlers := NewHandlers(services)
	workers := NewWorkers(redisClient, services, log)

	return &Container{
		Handlers: handlers,
		Workers:  workers,
		Redis:    redisClient,
		Logger:   log,
		Config:   cfg,
	}
}