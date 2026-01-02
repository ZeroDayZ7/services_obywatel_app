package service

import (
	"strconv"

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

// GetSessions pobiera listę sesji z JOINem do urządzeń
func (s *UserService) GetSessions(userIDStr string) ([]userModel.UserSessionDTO, error) {
	// 1. Konwersja string (z Gateway) na uint (dla bazy)
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return nil, err
	}

	// 2. Wywołujemy nową metodę z repozytorium auth
	return s.refreshRepo.GetSessionsWithDeviceData(uint(userID))
}

// RevokeSession unieważnia konkretną sesję
func (s *UserService) RevokeSession(userIDStr string, sessionIDStr string) error {
	// 1. Konwersja parametrów
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return err
	}
	sessionID, err := strconv.ParseUint(sessionIDStr, 10, 32)
	if err != nil {
		return err
	}

	// 2. Usunięcie sesji z bazy (Repo sprawdza czy sesja należy do tego UserID)
	return s.refreshRepo.DeleteByID(uint(sessionID), uint(userID))
}
