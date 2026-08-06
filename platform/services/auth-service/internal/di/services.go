package di

import (
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/service"
)

type Services struct {
	AuthService          service.AuthService
	UserService          service.UserService
	PasswordResetService service.PasswordResetService
	KeyService           service.KeyService
}

func NewServices(
	repos *Repositories,
	cache *redis.Cache,
	eventPublisher rabbitmq.EventPublisher,
	cfg *config.Config,
) *Services {
	return &Services{
		AuthService: service.NewAuthService(
			repos.UserRepo,
			repos.RefreshTokenRepo,
			cache,
			eventPublisher,
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
		KeyService: service.NewKeyService(cfg.JWT.KeyID, cfg.JWT.AccessPublicKey),
	}
}
