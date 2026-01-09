package service

import (
	"context"

	"github.com/google/uuid"
	authRepo "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/repository"
	userModel "github.com/zerodayz7/platform/services/auth-service/internal/features/users/model"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/users/repository"
)

type UserService struct {
	userRepo    repository.UserRepository
	refreshRepo authRepo.RefreshTokenRepository
}

func NewUserService(uRepo repository.UserRepository, rRepo authRepo.RefreshTokenRepository) *UserService {
	return &UserService{
		userRepo:    uRepo,
		refreshRepo: rRepo,
	}
}

// GetSessions
func (s *UserService) GetSessions(ctx context.Context, userID uuid.UUID) ([]userModel.UserSessionDTO, error) {
	return s.refreshRepo.GetSessionsWithDeviceData(ctx, userID)
}

// RevokeSession
func (s *UserService) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uint) error {
	return s.refreshRepo.DeleteByID(ctx, sessionID, userID)
}
