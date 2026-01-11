package di

import (
	"gorm.io/gorm"

	repo "github.com/zerodayz7/platform/services/auth-service/internal/repository"
	repoDB "github.com/zerodayz7/platform/services/auth-service/internal/repository/db"
)

type Repositories struct {
	UserRepo         repo.UserRepository
	RefreshTokenRepo repo.RefreshTokenRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		UserRepo:         repoDB.NewUserRepository(db),
		RefreshTokenRepo: repoDB.NewRefreshTokenRepository(db),
	}
}
