package service

import "github.com/zerodayz7/platform/services/auth-service/internal/features/users/repository"

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Tutaj później dodasz metody do zarządzania użytkownikami
