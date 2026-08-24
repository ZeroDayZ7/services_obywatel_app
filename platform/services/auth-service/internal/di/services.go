// internal/di/services.go
package di

import (
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/service"
)

type Services struct {
	AuthService          service.AuthService
	UserService          service.UserService
	PasswordResetService service.PasswordResetService
	ConsumerService      service.ConsumerService
}

func NewServices(
	repos *Repositories,
	cache *redis.Cache,
	cfg *config.Config,
) *Services {
	return &Services{
		AuthService: service.NewAuthService(
			repos.UserRepo,
			repos.EmployeeRepo,
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
		ConsumerService: service.NewConsumerService(repos.ConsumerRepo), // Używamy repozytorium z kontenera
	}
}
