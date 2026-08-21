package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/envelope"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/identity-service/internal/model"
	"github.com/zerodayz7/services/identity-service/internal/repository"
)

type CitizenService interface {
	RegisterCitizen(ctx context.Context, payload model.CitizenPayload) (*model.Citizen, error)
	GetCitizenByID(ctx context.Context, userID uuid.UUID) (*model.CitizenPayload, error)
}

type citizenService struct {
	repo       repository.CitizenRepository
	eventPub   rabbitmq.EventPublisher
	cryptor    *envelope.EnvelopeCryptor
	hmacSecret []byte
	keyAlias   string
}

func NewCitizenService(
	repo repository.CitizenRepository,
	eventPub rabbitmq.EventPublisher,
	cryptor *envelope.EnvelopeCryptor,
	hmacSecret []byte,
	keyAlias string,
) CitizenService {
	return &citizenService{
		repo:       repo,
		eventPub:   eventPub,
		cryptor:    cryptor,
		hmacSecret: hmacSecret,
		keyAlias:   keyAlias,
	}
}

func (s *citizenService) RegisterCitizen(ctx context.Context, payload model.CitizenPayload) (*model.Citizen, error) {
	userID := shared.NewUUIDv7()

	plaintextBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal citizen payload: %w", err)
	}

	encryptedPayload, err := s.cryptor.Seal(ctx, s.keyAlias, plaintextBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to seal citizen payload: %w", err)
	}

	peselHash := s.hashPESEL(payload.PESEL)

	citizen := &model.Citizen{
		UserID:        userID,
		PESELHash:     peselHash,
		EncryptedData: encryptedPayload.EncryptedData,
		EncryptedDEK:  encryptedPayload.EncryptedDEK,
		KeyVersion:    1,
	}

	auditLog := &model.CitizenAuditLog{
		ID:      shared.NewUUIDv7(),
		UserID:  userID,
		Action:  model.ActionCitizenRegistered,
		ActorID: userID,
	}

	if err := s.repo.CreateWithAudit(ctx, citizen, auditLog); err != nil {
		return nil, fmt.Errorf("failed to save citizen to repository: %w", err)
	}

	eventPayload, err := json.Marshal(map[string]any{
		"user_id":     citizen.UserID,
		"key_version": citizen.KeyVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	if err := s.eventPub.Publish(ctx, rabbitmq.TopicCitizenCreated, eventPayload); err != nil {
		return nil, fmt.Errorf("failed to publish citizen.created event: %w", err)
	}

	return citizen, nil
}

func (s *citizenService) GetCitizenByID(ctx context.Context, userID uuid.UUID) (*model.CitizenPayload, error) {
	citizen, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	encPayload := envelope.EncryptedPayload{
		EncryptedData: citizen.EncryptedData,
		EncryptedDEK:  citizen.EncryptedDEK,
	}

	plaintext, err := s.cryptor.Unseal(ctx, s.keyAlias, encPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to unseal citizen data: %w", err)
	}

	var payload model.CitizenPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal citizen payload: %w", err)
	}

	return &payload, nil
}

func (s *citizenService) hashPESEL(pesel string) string {
	h := hmac.New(sha256.New, s.hmacSecret)
	h.Write([]byte(pesel))
	return hex.EncodeToString(h.Sum(nil))
}
