package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"

	userModel "github.com/zerodayz7/platform/services/auth-service/internal/features/users/model"
)

type RefreshTokenRepository interface {
	Save(rt *model.RefreshToken) error
	Get(token string) (*model.RefreshToken, error)
	Revoke(token string) error
	GetByToken(token string) (*model.RefreshToken, error)
	Update(rt *model.RefreshToken) error

	GetSessionsWithDeviceData(ctx context.Context, userID uuid.UUID) ([]userModel.UserSessionDTO, error)
	DeleteByID(ctx context.Context, sessionID uint, userID uuid.UUID) error
	UpdateFingerprintByUser(
		userID uuid.UUID,
		oldFingerprint string,
		newFingerprint string,
	) error
}
