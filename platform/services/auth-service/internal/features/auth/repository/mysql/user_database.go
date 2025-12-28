package mysql

import (
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) GetByID(id uint) (*model.User, error) {
	var u model.User
	if err := r.DB.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Update(user *model.User) error {
	return r.DB.Save(user).Error
}
