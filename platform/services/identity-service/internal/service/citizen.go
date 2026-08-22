package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/envelope"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/security"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/identity-service/internal/model"
	"github.com/zerodayz7/services/identity-service/internal/repository"
)

type CitizenService interface {
	RegisterCitizen(ctx context.Context, payload model.CitizenPayload) (*model.RegisterCitizenResponse, error)
	GetCitizenByID(ctx context.Context, userID uuid.UUID) (*model.CitizenPayload, error)
}

type citizenService struct {
	repo       repository.CitizenRepository
	cryptor    *envelope.EnvelopeCryptor
	hmacSecret []byte
	keyAlias   string
}

func NewCitizenService(
	repo repository.CitizenRepository,
	cryptor *envelope.EnvelopeCryptor,
	hmacSecret []byte,
	keyAlias string,
) CitizenService {
	return &citizenService{
		repo:       repo,
		cryptor:    cryptor,
		hmacSecret: hmacSecret,
		keyAlias:   keyAlias,
	}
}

func (s *citizenService) RegisterCitizen(ctx context.Context, payload model.CitizenPayload) (*model.RegisterCitizenResponse, error) {
	peselHash := s.hashPESEL(payload.PESEL)

	// 1. Sprawdzamy czy obywatel istnieje w bazie
	existingCitizen, err := s.repo.GetByPESELHash(ctx, peselHash)
	if err == nil && existingCitizen != nil {
		return nil, repository.ErrCitizenAlreadyExists
	}

	// Jeśli błąd jest inny niż brak rekordu (sql.ErrNoRows), to coś poszło nie tak z bazą
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check citizen existence: %w", err)
	}

	userID := shared.NewUUIDv7()

	plaintextBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal citizen payload: %w", err)
	}

	encryptedPayload, err := s.cryptor.Seal(ctx, s.keyAlias, plaintextBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to seal citizen payload: %w", err)
	}

	// 1. Model Citizen
	citizen := &model.Citizen{
		UserID:        userID,
		PESELHash:     peselHash,
		EncryptedData: encryptedPayload.EncryptedData,
		EncryptedDEK:  encryptedPayload.EncryptedDEK,
		KeyVersion:    1,
	}

	// 2. Model UserAgreement
	agreementID := shared.NewUUIDv7()
	now := time.Now().UTC()
	agreementNumber := shared.GenerateAgreementNumber(now)

	agreement := &model.UserAgreement{
		ID:              agreementID,
		UserID:          userID,
		AgreementNumber: agreementNumber,
		PeselEncrypted:  encryptedPayload.EncryptedData,
		Status:          model.AgreementStatusActive,
		SignedAt:        now,
	}

	// 3. Generowanie PUK z platform/pkg/security
	rawPUK, err := security.GenerateOTP(8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PUK: %w", err)
	}
	pukHash := s.hashPESEL(rawPUK)

	puk := &model.UserPukCode{
		ID:              shared.NewUUIDv7(),
		UserAgreementID: agreementID,
		UserID:          userID,
		PukHash:         pukHash,
		Status:          model.PukStatusActive,
		FailedAttempts:  0,
		MaxAttempts:     3,
	}

	// 4. Model AuditLog
	auditLog := &model.CitizenAuditLog{
		ID:      shared.NewUUIDv7(),
		UserID:  userID,
		Action:  model.ActionCitizenRegistered,
		ActorID: userID,
	}

	// 5. Outbox Message dla RabbitMQ
	eventPayload, err := json.Marshal(map[string]any{
		"user_id":          citizen.UserID,
		"agreement_number": agreement.AgreementNumber,
		"key_version":      citizen.KeyVersion,
		"signed_at":        agreement.SignedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	outboxMsg := &model.OutboxMessage{
		ID:            shared.NewUUIDv7(),
		AggregateType: "citizen",
		AggregateID:   userID,
		EventType:     string(rabbitmq.TopicCitizenCreated),
		Payload:       eventPayload,
		Status:        model.OutboxStatusPending,
	}

	// Wykonanie transakcji w bazie
	err = s.repo.RegisterCitizenWorkflow(ctx, citizen, agreement, puk, auditLog, outboxMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to execute register citizen workflow: %w", err)
	}

	// Zwrot DTO z danymi dla urzędnika/frontendu
	return &model.RegisterCitizenResponse{
		UserID:          userID,
		AgreementNumber: agreementNumber,
		PukCode:         rawPUK,
		CreatedAt:       now,
	}, nil
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
