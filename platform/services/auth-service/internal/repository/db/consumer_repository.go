// internal/repository/db/consumer_repository.go
package db

import (
	"context"
	"fmt"

	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"gorm.io/gorm"
)

type ConsumerRepository interface {
	CreatePendingCitizen(ctx context.Context, user *model.User) error
}

type consumerRepository struct {
	db *gorm.DB
}

func NewConsumerRepository(db *gorm.DB) ConsumerRepository {
	return &consumerRepository{db: db}
}

func (r *consumerRepository) CreatePendingCitizen(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create pending citizen user account in db: %w", err)
	}
	return nil
}
