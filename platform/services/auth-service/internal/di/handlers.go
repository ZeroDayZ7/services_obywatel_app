package di

import (
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/types"
	authHandler "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/handler"
	userHandler "github.com/zerodayz7/platform/services/auth-service/internal/features/users/handler"
)

type Handlers struct {
	AuthHandler  *authHandler.AuthHandler
	ResetHandler *authHandler.ResetHandler
	UserHandler  *userHandler.UserHandler
}

func NewHandlers(services *Services, cache *redis.Cache, cfg *types.Config) *Handlers {
	return &Handlers{
		AuthHandler:  authHandler.NewAuthHandler(services.AuthService, cache, cfg),
		ResetHandler: authHandler.NewResetHandler(services.AuthService, cache),
		UserHandler:  userHandler.NewUserHandler(services.UserService),
	}
}
