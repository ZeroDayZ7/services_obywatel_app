package di

import (
	authHandler "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/handler"
	userHandler "github.com/zerodayz7/platform/services/auth-service/internal/features/users/handler"

	authService "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/service"
	userService "github.com/zerodayz7/platform/services/auth-service/internal/features/users/service"

	refRepo "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/repository/mysql"
	userRepo "github.com/zerodayz7/platform/services/auth-service/internal/features/users/repository/mysql"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"gorm.io/gorm"
)

// Container przechowuje wszystkie zależności serwisów i handlerów
type Container struct {
	AuthHandler *authHandler.AuthHandler
	UserHandler *userHandler.UserHandler
	Redis       *redis.Client
	Cache       *redis.Cache
}

// NewContainer tworzy nowy kontener z wszystkimi zależnościami
func NewContainer(db *gorm.DB, redisClient *redis.Client) *Container {
	// repozytorium użytkowników
	userRepo := userRepo.NewUserRepository(db)
	refreshRepo := refRepo.NewRefreshTokenRepository(db)
	// serwisy
	authSvc := authService.NewAuthService(userRepo, refreshRepo)
	userSvc := userService.NewUserService(userRepo)

	// Cache wrapper do sesji
	cache := redis.NewCache(redisClient, redis.SessionPrefix, config.AppConfig.SessionTTL)

	// handlery
	return &Container{
		AuthHandler: authHandler.NewAuthHandler(authSvc, cache),
		UserHandler: userHandler.NewUserHandler(userSvc),
		Redis:       redisClient,
		Cache:       cache,
	}
}
