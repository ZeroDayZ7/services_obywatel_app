package service

import (
	"context"

	"github.com/zerodayz7/services/identity-service/internal/repository"
)

type HealthService interface {
	CheckHealth(ctx context.Context) error
}

type healthService struct {
	repo repository.HealthRepository
}

func NewHealthService(repo repository.HealthRepository) HealthService {
	return &healthService{repo: repo}
}

func (s *healthService) CheckHealth(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
