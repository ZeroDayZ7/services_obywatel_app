package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	"github.com/zerodayz7/platform/pkg/envelope"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/security"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/storage"
	"github.com/zerodayz7/services/identity-service/internal/model"
	"github.com/zerodayz7/services/identity-service/internal/repository"
)

type CitizenService interface {
	RegisterCitizen(ctx context.Context, payload model.CitizenPayload) (*model.RegisterCitizenResponse, error)
	GetCitizenByID(ctx context.Context, userID uuid.UUID) (*model.CitizenPayload, error)
}

type citizenService struct {
	repo               repository.CitizenRepository
	cryptor            *envelope.EnvelopeCryptor
	storage            storage.StorageClient
	hmacSecret         []byte
	keyAlias           string
	agreementsKeyAlias string
}

func NewCitizenService(
	repo repository.CitizenRepository,
	cryptor *envelope.EnvelopeCryptor,
	storage storage.StorageClient,
	hmacSecret []byte,
	keyAlias string,
	agreementsKeyAlias string,
) CitizenService {
	return &citizenService{
		repo:               repo,
		cryptor:            cryptor,
		storage:            storage,
		hmacSecret:         hmacSecret,
		keyAlias:           keyAlias,
		agreementsKeyAlias: agreementsKeyAlias,
	}
}

// #region RegisterCitizen
func (s *citizenService) RegisterCitizen(ctx context.Context, payload model.CitizenPayload) (*model.RegisterCitizenResponse, error) {
	log := shared.GetLogger()
	peselHash := s.hashPESEL(payload.PESEL)

	existingCitizen, err := s.repo.GetByPESELHash(ctx, peselHash)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "DATABASE_ERROR",
			Type:    apperr.Internal,
			Message: "Wewnętrzny błąd bazy danych podczas weryfikacji obywatela.",
			Err:     err,
		}
	}
	if existingCitizen != nil {
		return nil, &apperr.AppError{
			Code:    "CITIZEN_EXISTS",
			Type:    apperr.Conflict,
			Message: "Obywatel z takim numerem PESEL już istnieje w systemie.",
		}
	}

	userID := shared.NewUUIDv7()
	plaintextBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "JSON_MARSHAL_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd przetwarzania danych żądania.",
			Err:     err,
		}
	}

	encryptedPayload, err := s.cryptor.Seal(ctx, s.keyAlias, plaintextBytes)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "ENCRYPTION_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd szyfrowania danych wrażliwych.",
			Err:     err,
		}
	}

	citizen := &model.Citizen{
		UserID:        userID,
		PESELHash:     peselHash,
		EncryptedData: encryptedPayload.EncryptedData,
		EncryptedDEK:  encryptedPayload.EncryptedDEK,
		KeyVersion:    encryptedPayload.KeyVersion,
	}

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
		VerifiedVia:     "SYSTEM",
	}

	rawPUK, err := security.GenerateOTP(8)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "PUK_GENERATION_FAILED",
			Type:    apperr.Internal,
			Message: "Nie udało się wygenerować kodu PUK.",
			Err:     err,
		}
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

	payloadSum := sha256.Sum256(plaintextBytes)
	payloadHash := hex.EncodeToString(payloadSum[:])
	clientIP := reqctx.GetIP(ctx)

	auditLog := &model.CitizenAuditLog{
		ID:          shared.NewUUIDv7(),
		UserID:      userID,
		Action:      model.ActionCitizenRegistered,
		ActorID:     userID,
		IPAddress:   clientIP,
		PayloadHash: payloadHash,
	}

	log.DebugObj("Created audit log", auditLog)

	eventPayload, err := json.Marshal(map[string]any{
		"user_id":          citizen.UserID,
		"agreement_number": agreement.AgreementNumber,
		"key_version":      citizen.KeyVersion,
		"signed_at":        agreement.SignedAt,
	})
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "EVENT_MARSHAL_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd serializacji zdarzenia systemowego.",
			Err:     err,
		}
	}

	outboxMsg := &model.OutboxMessage{
		ID:            shared.NewUUIDv7(),
		AggregateType: "citizen",
		AggregateID:   userID,
		EventType:     string(rabbitmq.TopicCitizenCreated),
		Payload:       eventPayload,
		Status:        model.OutboxStatusPending,
	}

	// Atomowe wykonanie zapisu w ramach jednej transakcji
	err = s.repo.WithinTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, citizen); err != nil {
			return fmt.Errorf("create citizen: %w", err)
		}
		if err := s.repo.CreateAgreement(txCtx, agreement); err != nil {
			return fmt.Errorf("create agreement: %w", err)
		}
		if err := s.repo.CreatePukCode(txCtx, puk); err != nil {
			return fmt.Errorf("create PUK: %w", err)
		}
		if err := s.repo.CreateAuditLog(txCtx, auditLog); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}
		if err := s.repo.CreateOutboxMessage(txCtx, outboxMsg); err != nil {
			return fmt.Errorf("create outbox message: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "WORKFLOW_FAILED",
			Type:    apperr.Internal,
			Message: "Nie udało się zapisać danych obywatela w bazie.",
			Err:     err,
		}
	}

	return &model.RegisterCitizenResponse{
		UserID:          userID,
		AgreementNumber: agreementNumber,
		PukCode:         rawPUK,
		CreatedAt:       now,
	}, nil
}

// #region GetCitizenByID
func (s *citizenService) GetCitizenByID(ctx context.Context, userID uuid.UUID) (*model.CitizenPayload, error) {
	citizen, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "DATABASE_ERROR",
			Type:    apperr.Internal,
			Message: "Błąd podczas pobierania danych obywatela.",
			Err:     err,
		}
	}
	if citizen == nil {
		return nil, apperr.ErrNotFound
	}

	encPayload := envelope.EncryptedPayload{
		EncryptedData: citizen.EncryptedData,
		EncryptedDEK:  citizen.EncryptedDEK,
	}

	plaintext, err := s.cryptor.Unseal(ctx, s.keyAlias, encPayload)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "DECRYPTION_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd deszyfrowania danych obywatela.",
			Err:     err,
		}
	}

	var payload model.CitizenPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, &apperr.AppError{
			Code:    "JSON_UNMARSHAL_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd parsowania odszyfrowanych danych.",
			Err:     err,
		}
	}

	return &payload, nil
}

func (s *citizenService) GenerateAndSaveAgreement(ctx context.Context, userID uuid.UUID, pdfBytes []byte) (*model.UserAgreement, error) {
	encryptedPayload, err := s.cryptor.Seal(ctx, s.agreementsKeyAlias, pdfBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt agreement pdf: %w", err)
	}

	// 2. Generujemy unikalną ścieżkę w S3
	agreementID := shared.NewUUIDv7()

	// Ścieżka: agreements/users/{user_id}/{agreement_v7_id}.pdf.enc
	filename := fmt.Sprintf("agreements/users/%s/%s.pdf.enc", userID.String(), agreementID.String())

	// 3. Wysyłamy ZASZYFROWANE bajty do S3
	reader := bytes.NewReader(encryptedPayload.EncryptedData)
	s3Key, err := s.storage.Upload(ctx, filename, reader, int64(len(encryptedPayload.EncryptedData)), "application/octet-stream")
	if err != nil {
		return nil, fmt.Errorf("failed to upload encrypted agreement to S3: %w", err)
	}

	// 4. Budujemy obiekt do zapisu w bazie danych
	agreement := &model.UserAgreement{
		ID:           shared.NewUUIDv7(),
		UserID:       userID,
		S3Key:        s3Key,
		S3Bucket:     "citizens-data",
		EncryptedDEK: encryptedPayload.EncryptedDEK,
		KeyVersion:   encryptedPayload.KeyVersion,
		Status:       model.AgreementStatusPending,
		SignedAt:     time.Now(),
	}

	// 5. Zapisujemy w repozytorium...
	return agreement, nil
}

// #region hashPESEL
func (s *citizenService) hashPESEL(pesel string) string {
	h := hmac.New(sha256.New, s.hmacSecret)
	h.Write([]byte(pesel))
	return hex.EncodeToString(h.Sum(nil))
}
