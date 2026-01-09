package di

import (
	authService "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/service"
	userService "github.com/zerodayz7/platform/services/auth-service/internal/features/users/service"
)

type Services struct {
	AuthService *authService.AuthService
	UserService *userService.UserService
}

func NewServices(repos *Repositories) *Services {
	return &Services{
		AuthService: authService.NewAuthService(
			repos.UserRepo,
			repos.RefreshTokenRepo,
		),
		UserService: userService.NewUserService(
			repos.UserRepo,
			repos.RefreshTokenRepo,
		),
	}
}
