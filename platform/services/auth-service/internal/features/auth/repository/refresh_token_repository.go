package repository

import (
	"time"

	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
)

type RefreshTokenRepository interface {
	Save(userID uint, token string, duration time.Duration) error
	Get(token string) (*model.RefreshToken, error)
	Revoke(token string) error
	GetByToken(token string) (*model.RefreshToken, error)
	Update(rt *model.RefreshToken) error
}
