package di

import (
	"github.com/zerodayz7/platform/pkg/types"
	authService "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/service"
	userService "github.com/zerodayz7/platform/services/auth-service/internal/features/users/service"
)

type Services struct {
	AuthService *authService.AuthService
	UserService *userService.UserService
}

func NewServices(repos *Repositories, cfg *types.Config) *Services {
	return &Services{
		AuthService: authService.NewAuthService(
			repos.UserRepo,
			repos.RefreshTokenRepo,
			cfg,
		),
		UserService: userService.NewUserService(
			repos.UserRepo,
			repos.RefreshTokenRepo,
		),
	}
}
