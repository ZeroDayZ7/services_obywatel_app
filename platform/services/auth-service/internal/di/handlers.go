package di

import (
	authHandler "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/handler"
	userHandler "github.com/zerodayz7/platform/services/auth-service/internal/features/users/handler"

	"github.com/zerodayz7/platform/pkg/redis"
)

type Handlers struct {
	AuthHandler  *authHandler.AuthHandler
	ResetHandler *authHandler.ResetHandler
	UserHandler  *userHandler.UserHandler
}

func NewHandlers(services *Services, cache *redis.Cache) *Handlers {
	return &Handlers{
		AuthHandler:  authHandler.NewAuthHandler(services.AuthService, cache),
		ResetHandler: authHandler.NewResetHandler(services.AuthService, cache),
		UserHandler:  userHandler.NewUserHandler(services.UserService),
	}
}
