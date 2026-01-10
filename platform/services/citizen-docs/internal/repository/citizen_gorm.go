package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"gorm.io/gorm"
)

type citizenRepository struct {
	db *gorm.DB
}

func NewCitizenRepository(db *gorm.DB) CitizenRepo {
	return &citizenRepository{db: db}
}

func (r *citizenRepository) Create(ctx context.Context, profile *model.CitizenProfile) error {
	// Używamy WithContext, aby GORM wiedział o cancelation/timeoutach
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *citizenRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.CitizenProfile, error) {
	var profile model.CitizenProfile
	err := r.db.WithContext(ctx).
		Preload("Documents").
		Where("user_id = ?", userID).
		First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *citizenRepository) GetByPeselHash(ctx context.Context, hash string) (*model.CitizenProfile, error) {
	var profile model.CitizenProfile
	err := r.db.WithContext(ctx).Where("pesel_hash = ?", hash).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}
