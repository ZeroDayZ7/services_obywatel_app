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
	AuthHandler  *authHandler.AuthHandler
	ResetHandler *authHandler.ResetHandler
	UserHandler  *userHandler.UserHandler
	Redis        *redis.Client
	Cache        *redis.Cache
}

// NewContainer tworzy nowy kontener z wszystkimi zależnościami
func NewContainer(db *gorm.DB, redisClient *redis.Client) *Container {
	// 1. Inicjalizacja Repozytoriów
	uRepo := userRepo.NewUserRepository(db)
	rRepo := refRepo.NewRefreshTokenRepository(db)

	// 2. Inicjalizacja Serwisów
	// authSvc potrzebuje obu repozytoriów do logowania i odświeżania
	authSvc := authService.NewAuthService(uRepo, rRepo)

	// userSvc teraz również potrzebuje rRepo, aby pobierać i usuwać sesje
	userSvc := userService.NewUserService(uRepo, rRepo)

	// 3. Konfiguracja Cache (Redis)
	cache := redis.NewCache(redisClient, redis.SessionPrefix, config.AppConfig.SessionTTL)

	// 4. Inicjalizacja Handlerów
	authH := authHandler.NewAuthHandler(authSvc, cache)
	resetH := authHandler.NewResetHandler(authSvc, cache)
	userH := userHandler.NewUserHandler(userSvc, rRepo)

	return &Container{
		AuthHandler:  authH,
		ResetHandler: resetH,
		UserHandler:  userH,
		Redis:        redisClient,
		Cache:        cache,
	}
}
