package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"

	userModel "github.com/zerodayz7/platform/services/auth-service/internal/features/users/model"
)

type RefreshTokenRepository interface {
	Save(userID uuid.UUID, token string, fingerprint string, ttl time.Duration) error
	Get(token string) (*model.RefreshToken, error)
	Revoke(token string) error
	GetByToken(token string) (*model.RefreshToken, error)
	Update(rt *model.RefreshToken) error

	GetSessionsWithDeviceData(userID uuid.UUID) ([]userModel.UserSessionDTO, error)
	DeleteByID(sessionID uint, userID uuid.UUID) error
	UpdateFingerprintByUser(
		userID uuid.UUID,
		oldFingerprint string,
		newFingerprint string,
	) error
}

