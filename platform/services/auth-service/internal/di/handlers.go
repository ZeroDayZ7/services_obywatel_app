package di

import (
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/viper"

	handler "github.com/zerodayz7/platform/services/auth-service/internal/handler"
)

type Handlers struct {
	AuthHandler  *handler.AuthHandler
	ResetHandler *handler.ResetHandler
	UserHandler  *handler.UserHandler
}

func NewHandlers(services *Services, cache *redis.Cache, cfg *viper.Config) *Handlers {
	return &Handlers{
		AuthHandler:  handler.NewAuthHandler(services.AuthService, cache, cfg),
		ResetHandler: handler.NewResetHandler(services.PasswordResetService, cache),
		UserHandler:  handler.NewUserHandler(services.UserService),
	}
}
