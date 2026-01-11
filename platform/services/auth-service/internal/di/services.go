package di

import (
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/auth-service/internal/service"
)

type Services struct {
	AuthService          service.AuthService
	UserService          service.UserService
	PasswordResetService service.PasswordResetService
}

func NewServices(repos *Repositories, cache *redis.Cache, cfg *viper.Config) *Services {
	return &Services{
		AuthService: service.NewAuthService(
			repos.UserRepo,
			repos.RefreshTokenRepo,
			cache,
			cfg,
		),
		UserService: service.NewUserService(
			repos.UserRepo,
			repos.RefreshTokenRepo,
		),
		PasswordResetService: service.NewPasswordResetService(
			repos.UserRepo,
			repos.RefreshTokenRepo,
			cache,
		),
	}
}
