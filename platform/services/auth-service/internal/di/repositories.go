package di

import (
	authRepo "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/repository"
	authRepoDB "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/repository/db"

	userRepo "github.com/zerodayz7/platform/services/auth-service/internal/features/users/repository"
	userRepoDB "github.com/zerodayz7/platform/services/auth-service/internal/features/users/repository/db"

	"gorm.io/gorm"
)

type Repositories struct {
	UserRepo         userRepo.UserRepository
	RefreshTokenRepo authRepo.RefreshTokenRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		UserRepo:         userRepoDB.NewUserRepository(db),
		RefreshTokenRepo: authRepoDB.NewRefreshTokenRepository(db),
	}
}
