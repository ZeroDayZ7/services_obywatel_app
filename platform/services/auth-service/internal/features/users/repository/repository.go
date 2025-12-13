package repository

import "github.com/zerodayz7/platform/services/auth-service/internal/features/users/model"

type UserRepository interface {
	CreateUser(*model.User) error
	GetByID(uint) (*model.User, error)
	GetByEmail(string) (*model.User, error)
	EmailExists(string) (bool, error)
	UsernameExists(string) (bool, error)
	EmailOrUsernameExists(email, username string) (bool, bool, error)
}
