package service

import (
	"fmt"
	"strconv"

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

// GetSessions pobiera listę sesji z JOINem do urządzeń
func (s *UserService) GetSessions(userIDStr string) ([]userModel.UserSessionDTO, error) {
	// 1. Konwersja string (z Gateway) na uint (dla bazy)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		// Zwracamy błąd, bo userIDStr nie jest poprawnym formatem UUID
		return nil, err
	}

	// 2. Wywołujemy nową metodę z repozytorium auth
	return s.refreshRepo.GetSessionsWithDeviceData(uuid.UUID(userID))
}

// RevokeSession unieważnia konkretną sesję
func (s *UserService) RevokeSession(userIDStr string, sessionIDStr string) error {
	// 1. Konwersja userID na UUID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user id format: %w", err)
	}

	// 2. Konwersja sessionID na uint (zakładając, że to ID rekordu w bazie)
	sessionID, err := strconv.ParseUint(sessionIDStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid session id format: %w", err)
	}

	// 3. Usunięcie sesji
	// userID to już uuid.UUID, więc nie musisz go rzutować
	return s.refreshRepo.DeleteByID(uint(sessionID), userID)
}


