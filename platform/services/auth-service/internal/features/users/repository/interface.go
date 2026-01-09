package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
)

type UserRepository interface {
	CreateUser(*model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	EmailExists(string) (bool, error)
	UsernameExists(string) (bool, error)
	EmailOrUsernameExists(email, username string) (bool, bool, error)
	Update(ctx context.Context, user *model.User) error
	SaveDevice(ctx context.Context, device *model.UserDevice) error
	UpdateFailedLogin(userID uuid.UUID, attempts int) error

	GetDeviceByFingerprint(ctx context.Context, userID uuid.UUID, fingerprint string) (*model.UserDevice, error)
	UpdateDeviceStatus(ctx context.Context, deviceID uuid.UUID, publicKey string, deviceName string, isActive bool, isVerified bool) error
}
