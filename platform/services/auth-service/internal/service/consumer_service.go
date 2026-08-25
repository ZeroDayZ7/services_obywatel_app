// internal/service/consumer_service.go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	repo "github.com/zerodayz7/platform/services/auth-service/internal/repository" // DODANE: import interfejsu repozytorium
)

type ConsumerService interface {
	CreateCitizenAccountFromEvent(ctx context.Context, citizenID uuid.UUID, agreementNumber string) error
}

type consumerService struct {
	consumerRepo repo.ConsumerRepository
	log          *shared.Logger
}

//#region NewConsumerService
func NewConsumerService(consumerRepo repo.ConsumerRepository) ConsumerService {
	return &consumerService{
		consumerRepo: consumerRepo,
		log:          shared.GetLogger(),
	}
}

//#region CreateCitizenAccountFromEvent
func (s *consumerService) CreateCitizenAccountFromEvent(ctx context.Context, citizenID uuid.UUID, agreementNumber string) error {
	s.log.Info("🔄 Tworzenie konta obywatela na podstawie eventu z RabbitMQ",
		"citizen_id", citizenID,
		"agreement_number", agreementNumber,
	)

	tempUsername := fmt.Sprintf("citizen_%s", citizenID.String()[:8])
	tempEmail := fmt.Sprintf("pending_%s@citizen.local", citizenID.String()[:8])
	dummyPassword := fmt.Sprintf("LOCKED_PENDING_ACCOUNT_%s", uuid.New().String())

	user := &model.User{
		ID:        citizenID,
		Username:  tempUsername,
		Email:     tempEmail,
		Password:  dummyPassword,
		Role:      model.RoleCitizen,
		Status:    model.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.consumerRepo.CreatePendingCitizen(ctx, user); err != nil {
		s.log.Error("❌ Nie udało się utworzyć konta PENDING dla obywatela w DB", "error", err, "citizen_id", citizenID)
		return err
	}

	s.log.Info("✅ Pomyślnie utworzono konto PENDING dla obywatela", "user_id", citizenID, "username", tempUsername)
	return nil
}
