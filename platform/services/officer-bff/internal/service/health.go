package service

import "context"

type HealthService interface {
	CheckHealth(ctx context.Context) error
}

type healthService struct{}

func NewHealthService() HealthService {
	return &healthService{}
}

func (s *healthService) CheckHealth(ctx context.Context) error {
	return nil
}
