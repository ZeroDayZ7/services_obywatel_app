package repository

import (
	"time"

	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
	// DODAJ TEN IMPORT TUTAJ:
	userModel "github.com/zerodayz7/platform/services/auth-service/internal/features/users/model"
)

type UserRepository interface {
	GetByID(id uint) (*model.User, error)
	Update(user *model.User) error
}

type RefreshTokenRepository interface {
	Save(userID uint, token string, fingerprint string, ttl time.Duration) error
	Get(token string) (*model.RefreshToken, error)
	Revoke(token string) error
	GetByToken(token string) (*model.RefreshToken, error)
	Update(rt *model.RefreshToken) error

	// Teraz userModel będzie już rozpoznawany
	GetSessionsWithDeviceData(userID uint) ([]userModel.UserSessionDTO, error)
	DeleteByID(sessionID uint, userID uint) error
	UpdateFingerprintByUser(
		userID uint,
		oldFingerprint string,
		newFingerprint string,
	) error
}
