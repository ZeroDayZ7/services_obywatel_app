package di

import (
	"github.com/zerodayz7/platform/pkg/redis"

	"github.com/zerodayz7/platform/services/auth-service/config"
	handler "github.com/zerodayz7/platform/services/auth-service/internal/handler"
)

type Handlers struct {
	AuthHandler      *handler.AuthHandler
	ResetHandler     *handler.ResetHandler
	UserHandler      *handler.UserHandler
	WellKnownHandler *handler.WellKnownHandler
}

func NewHandlers(services *Services, cache *redis.Cache, cfg *config.Config) *Handlers {
	return &Handlers{
		AuthHandler:      handler.NewAuthHandler(services.AuthService, cache, cfg),
		ResetHandler:     handler.NewResetHandler(services.PasswordResetService, cache),
		UserHandler:      handler.NewUserHandler(services.UserService),
		WellKnownHandler: handler.NewWellKnownHandler(services.KeyService),
	}
}
