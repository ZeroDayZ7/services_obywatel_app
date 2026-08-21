package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"github.com/zerodayz7/platform/services/auth-service/internal/repository"
	"gorm.io/gorm"
)

type employeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) repository.EmployeeRepository {
	return &employeeRepository{db: db}
}

func (r *employeeRepository) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*model.EmployeeProfile, error) {
	var profile model.EmployeeProfile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("employee profile not found")
		}
		return nil, err
	}
	return &profile, nil
}

func (r *employeeRepository) GetActiveCredentialByUserID(ctx context.Context, userID uuid.UUID) (*model.EmployeeCredential, error) {
	var credential model.EmployeeCredential
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, model.EmployeeCredentialActive).
		First(&credential).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("active employee credential not found")
		}
		return nil, err
	}
	return &credential, nil
}
