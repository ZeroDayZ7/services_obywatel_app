package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
)

type UserRepository interface {
	CreateUser(*model.User) error
	GetByID(uuid.UUID) (*model.User, error)
	GetByEmail(string) (*model.User, error)
	EmailExists(string) (bool, error)
	UsernameExists(string) (bool, error)
	EmailOrUsernameExists(email, username string) (bool, bool, error)
	Update(user *model.User) error
	SaveDevice(ctx context.Context, device *model.UserDevice) error
	UpdateFailedLogin(userID uuid.UUID, attempts int) error
}
