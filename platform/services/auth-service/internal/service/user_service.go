package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"github.com/zerodayz7/platform/services/auth-service/internal/repository"
)

type UserService interface {
	// Dodano fingerprint jako trzeci parametr
	GetSessions(ctx context.Context, userID uuid.UUID, fingerprint string) ([]model.UserSessionDTO, error)
	RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uint) error
}

type userService struct {
	userRepo    repository.UserRepository
	refreshRepo repository.RefreshTokenRepository
}

func NewUserService(uRepo repository.UserRepository, rRepo repository.RefreshTokenRepository) UserService {
	return &userService{
		userRepo:    uRepo,
		refreshRepo: rRepo,
	}
}

// Implementacja uwzględniająca fingerprint
func (s *userService) GetSessions(ctx context.Context, userID uuid.UUID, fingerprint string) ([]model.UserSessionDTO, error) {
	sessions, err := s.refreshRepo.GetSessions(ctx, userID) // Wywołanie repo (2 argumenty)
	if err != nil {
		return nil, err
	}

	for i := range sessions {
		if sessions[i].Fingerprint == fingerprint {
			sessions[i].IsCurrent = true
		}
	}
	return sessions, nil
}

func (s *userService) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uint) error {
	return s.refreshRepo.RevokeSession(ctx, userID, sessionID)
}
