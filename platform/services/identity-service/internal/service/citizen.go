package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

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
	agreement := &model.UserAgreement{
		ID:              agreementID,
		UserID:          userID,
		AgreementNumber: fmt.Sprintf("AGR/%s/%d", now.Format("20060102"), now.Unix()%100000),
		PeselEncrypted:  encryptedPayload.EncryptedData, // Lub wyizolowane zaszyfrowane pole PESEL
		Status:          model.AgreementStatusActive,
		SignedAt:        now,
	}

	// 3. Model UserPukCode (Generowanie np. 8-cyfrowego PUK)
	rawPUK, err := generateRandomPUK(8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PUK: %w", err)
	}
	pukHash := s.hashPESEL(rawPUK) // Hashowanie kodu PUK

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

	// 5. Model OutboxMessage (zamiast bezpośredniej publikacji RabbitMQ)
	eventPayload, err := json.Marshal(map[string]any{
		"user_id":     citizen.UserID,
		"key_version": citizen.KeyVersion,
		"signed_at":   agreement.SignedAt,
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

	// Wykonanie całej operacji w jednej transakcji DB
	err = s.repo.RegisterCitizenWorkflow(ctx, citizen, agreement, puk, auditLog, outboxMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to execute register citizen workflow: %w", err)
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

func generateRandomPUK(length int) (string, error) {
	const digits = "0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		result[i] = digits[num.Int64()]
	}
	return string(result), nil
}
