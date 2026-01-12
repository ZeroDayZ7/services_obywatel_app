package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
)

type RefreshTokenRepository interface {
	Save(rt *model.RefreshToken) error
	Get(token string) (*model.RefreshToken, error)
	Revoke(token string) error
	GetByToken(token string) (*model.RefreshToken, error)
	Update(rt *model.RefreshToken) error
	UpdateFingerprintByUser(userID uuid.UUID, oldFingerprint string, newFingerprint string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	GetSessions(ctx context.Context, userID uuid.UUID) ([]model.UserSessionDTO, error)
	RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uint) error
}

type UserRepository interface {
	CreateUser(*model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)

	EmailExists(string) (bool, error)
	UsernameExists(string) (bool, error)
	EmailOrUsernameExists(email, username string) (bool, bool, error)
	Update(ctx context.Context, user *model.User) error
	SaveDevice(ctx context.Context, device *model.UserDevice) error

	// Dopasuj te nazwy dokładnie do tego, co wywołujesz w AuthService
	IncrementUserFailedLogin(userID uuid.UUID) error
	ResetFailedLoginAttempts(userID uuid.UUID) error

	GetDeviceByFingerprint(ctx context.Context, userID uuid.UUID, fingerprint string) (*model.UserDevice, error)
}
