package mysql

import (
	"errors"

	"github.com/zerodayz7/platform/services/auth-service/internal/features/users/model"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/users/repository"
	"gorm.io/gorm"
)

var _ repository.UserRepository = (*MySQLUserRepo)(nil)

type MySQLUserRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *MySQLUserRepo {
	return &MySQLUserRepo{db: db}
}

func (r *MySQLUserRepo) CreateUser(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *MySQLUserRepo) GetByID(id uint) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *MySQLUserRepo) GetByEmail(email string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *MySQLUserRepo) EmailExists(email string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *MySQLUserRepo) UsernameExists(username string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *MySQLUserRepo) EmailOrUsernameExists(email, username string) (emailExists, usernameExists bool, err error) {
	var users []model.User
	if err := r.db.Where("email = ? OR username = ?", email, username).Find(&users).Error; err != nil {
		return false, false, err
	}

	for _, u := range users {
		if u.Email == email {
			emailExists = true
		}
		if u.Username == username {
			usernameExists = true
		}
	}
	return
}
