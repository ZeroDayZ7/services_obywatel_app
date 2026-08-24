package service

import (
	"bytes"
	"context"
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

// #region Interface
type CitizenService interface {
	RegisterCitizen(ctx context.Context, payload model.CitizenPayload) (*model.RegisterCitizenResponse, error)
	GetCitizenByID(ctx context.Context, userID uuid.UUID) (*model.CitizenPayload, error)
	DownloadAgreementPDF(ctx context.Context, agreementID uuid.UUID) ([]byte, error)
}

type citizenService struct {
	repo               repository.CitizenRepository
	cryptor            *envelope.EnvelopeCryptor
	storage            storage.StorageClient
	pdfGen             PDFGenerator
	hmacPeselSecret    []byte
	hmacPhoneSecret    []byte
	hmacEmailSecret    []byte
	hmacPukSecret      []byte
	dataKeyAlias       string
	agreementsKeyAlias string
	phoneKeyAlias      string
	emailKeyAlias      string
}

func NewCitizenService(
	repo repository.CitizenRepository,
	cryptor *envelope.EnvelopeCryptor,
	storage storage.StorageClient,
	pdfGen PDFGenerator,
	hmacPeselSecret []byte,
	hmacPhoneSecret []byte,
	hmacEmailSecret []byte,
	hmacPukSecret []byte,
	dataKeyAlias string,
	agreementsKeyAlias string,
	phoneKeyAlias string,
	emailKeyAlias string,
) CitizenService {
	return &citizenService{
		repo:               repo,
		cryptor:            cryptor,
		storage:            storage,
		pdfGen:             pdfGen,
		hmacPeselSecret:    hmacPeselSecret,
		hmacPhoneSecret:    hmacPhoneSecret,
		hmacEmailSecret:    hmacEmailSecret,
		hmacPukSecret:      hmacPukSecret,
		dataKeyAlias:       dataKeyAlias,
		agreementsKeyAlias: agreementsKeyAlias,
		phoneKeyAlias:      phoneKeyAlias,
		emailKeyAlias:      emailKeyAlias,
	}
}

// #region RegisterCitizen
func (s *citizenService) RegisterCitizen(ctx context.Context, payload model.CitizenPayload) (*model.RegisterCitizenResponse, error) {
	log := shared.GetLogger()
	log.Debug("🔍 Przed obliczeniem haszy",
		"pesel_raw_len", len(payload.PESEL),
		"email_raw_len", len(payload.Email),
		"phone_raw_len", len(payload.PhoneNumber),
	)

	peselHash := hex.EncodeToString(reqctx.ComputeMAC([]byte(payload.PESEL), s.hmacPeselSecret))
	emailHash := hex.EncodeToString(reqctx.ComputeMAC([]byte(payload.Email), s.hmacEmailSecret))
	phoneHash := hex.EncodeToString(reqctx.ComputeMAC([]byte(payload.PhoneNumber), s.hmacPhoneSecret))

	log.Debug("✅ Po obliczeniu haszy",
		"pesel_hash", peselHash,
		"email_hash", emailHash,
		"phone_hash", phoneHash,
	)

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

	// 1. Szyfrowanie danych wrażliwych obywatela
	encryptedPayload, err := s.cryptor.Seal(ctx, s.dataKeyAlias, plaintextBytes)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "ENCRYPTION_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd szyfrowania danych wrażliwych.",
			Err:     err,
		}
	}

	// 2. Niezależne szyfrowanie numeru telefonu (do 2FA)
	phoneEncryptedPayload, err := s.cryptor.Seal(ctx, s.phoneKeyAlias, []byte(payload.PhoneNumber))
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "PHONE_ENCRYPTION_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd szyfrowania numeru telefonu.",
			Err:     err,
		}
	}

	// 3. Niezależne szyfrowanie adresu e-mail (do 2FA / powiadomień)
	emailEncryptedPayload, err := s.cryptor.Seal(ctx, s.emailKeyAlias, []byte(payload.Email))
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "EMAIL_ENCRYPTION_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd szyfrowania adresu e-mail.",
			Err:     err,
		}
	}

	actorID := reqctx.GetUserID(ctx)
	if actorID == uuid.Nil {
		actorID = userID
	}

	citizen := &model.Citizen{
		UserID:        userID,
		PESELHash:     peselHash,
		EmailHash:     emailHash,
		PhoneHash:     phoneHash,
		EncryptedData: encryptedPayload.EncryptedData,
		EncryptedDEK:  encryptedPayload.EncryptedDEK,
		KeyVersion:    encryptedPayload.KeyVersion,
	}

	agreementID := shared.NewUUIDv7()
	now := time.Now().UTC()
	agreementNumber := shared.GenerateAgreementNumber(now)

	var secondNamePtr *string
	if payload.SecondName != nil && *payload.SecondName != "" {
		secondNamePtr = payload.SecondName
	}

	var flatNumPtr *string
	if payload.FlatNumber != nil && *payload.FlatNumber != "" {
		flatNumPtr = payload.FlatNumber
	}

	// Najpierw generujemy wstępny PDF lub potrzebujemy hasha
	// Obliczamy hash dokumentu (np. z zaszyfrowanego payloadu danych lub samej struktury)
	docHashBytes := sha256.Sum256(plaintextBytes)
	documentHash := hex.EncodeToString(docHashBytes[:])

	reqCtx, ok := reqctx.FromContext(ctx)

	officerName := "System Automatyczny"
	departmentIDStr := "-"
	institutionIDStr := "-"

	if ok && reqCtx != nil && reqCtx.Role == "OFFICER" {
		if reqCtx.Username != "" {
			officerName = reqCtx.Username
		}
		if reqCtx.DepartmentID != nil {
			departmentIDStr = reqCtx.DepartmentID.String()
		}
		if reqCtx.InstitutionID != nil {
			institutionIDStr = reqCtx.InstitutionID.String()
		}
	}

	templateData := AgreementTemplateData{
		AgreementID:     agreementID.String(),
		AgreementNumber: agreementNumber,
		FirstName:       payload.FirstName,
		SecondName:      secondNamePtr,
		LastName:        payload.LastName,
		PESEL:           payload.PESEL,
		Email:           payload.Email,
		Street:          payload.Street,
		HouseNumber:     payload.HouseNumber,
		FlatNumber:      flatNumPtr,
		PostalCode:      payload.PostalCode,
		City:            payload.City,
		PhoneNumber:     payload.PhoneNumber,
		SignedAt:        now.Format("02.01.2006 15:04"),
		KeyVersion:      int(encryptedPayload.KeyVersion),
		DocumentHash:    documentHash,
		OfficerName:     officerName,
		OfficerID:       actorID.String(),
		DepartmentID:    departmentIDStr,
		InstitutionID:   institutionIDStr,
	}

	log.DebugJSON("Pełny payload danych do generowania umowy PDF", templateData)

	pdfBytes, err := s.pdfGen.GenerateAgreementPDF(ctx, templateData)
	if err != nil {
		log.Debug("[ERROR] PDF Generation failed: %v", err)
		return nil, &apperr.AppError{
			Code:    "PDF_GENERATION_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd generowania dokumentu umowy.",
			Err:     err,
		}
	}

	// 3. Szyfrowanie pliku PDF umowy (Envelope Encryption)
	pdfEncryptedPayload, err := s.cryptor.Seal(ctx, s.agreementsKeyAlias, pdfBytes)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "PDF_ENCRYPTION_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd szyfrowania pliku umowy.",
			Err:     err,
		}
	}

	// 4. Upload zaszyfrowanego PDF do MinIO / S3
	s3Bucket := "agreements"
	s3Key := fmt.Sprintf("users/%s/%s.pdf.enc", userID.String(), agreementID.String())

	pdfReader := bytes.NewReader(pdfEncryptedPayload.EncryptedData)
	pdfSize := int64(len(pdfEncryptedPayload.EncryptedData))

	_, err = s.storage.Upload(ctx, s3Key, pdfReader, pdfSize, "application/octet-stream")
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "STORAGE_UPLOAD_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd zapisu pliku umowy w magazynie danych.",
			Err:     err,
		}
	}

	// 5. Generowanie czasowego presigned URL do pobrania (np. 15 minut)
	downloadURL, err := s.storage.GetPresignedURL(ctx, s3Key, 15*time.Minute)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "STORAGE_PRESIGN_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd generowania linku pobierania umowy.",
			Err:     err,
		}
	}

	// 6. Przygotowanie struktury UserAgreement z osobnym kluczem DEK dla pliku PDF
	agreement := &model.UserAgreement{
		ID:              agreementID,
		UserID:          userID,
		AgreementNumber: agreementNumber,
		S3Key:           s3Key,
		S3Bucket:        s3Bucket,
		EncryptedDEK:    pdfEncryptedPayload.EncryptedDEK,
		KeyVersion:      pdfEncryptedPayload.KeyVersion,
		EncryptedEmail:  emailEncryptedPayload.EncryptedData,
		EncryptedPhone:  phoneEncryptedPayload.EncryptedData,
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

	pukHash := hex.EncodeToString(reqctx.ComputeMAC([]byte(rawPUK), s.hmacPukSecret))

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
		ActorID:     actorID,
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

	// 7. Atomowy zapis w bazie danych w transakcji
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
		log.Error("Workflow transaction failed", "err", err)
		return nil, &apperr.AppError{
			Code:    "WORKFLOW_FAILED",
			Type:    apperr.Internal,
			Message: "Nie udało się zapisać danych obywatela w bazie.",
			Err:     err,
		}
	}

	downloadURL = fmt.Sprintf("/agreements/%s/download", agreementID.String())

	return &model.RegisterCitizenResponse{
		UserID:               userID,
		AgreementNumber:      agreementNumber,
		AgreementDownloadURL: downloadURL,
		PukCode:              rawPUK,
		CreatedAt:            now,
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

	plaintext, err := s.cryptor.Unseal(ctx, s.dataKeyAlias, encPayload)
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

// #region GenerateAndSaveAgreement
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

// #region DownloadAgreementPDF
func (s *citizenService) DownloadAgreementPDF(ctx context.Context, agreementID uuid.UUID) ([]byte, error) {
	// log := shared.GetLogger()
	agreement, err := s.repo.GetAgreementByID(ctx, agreementID)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "DATABASE_ERROR",
			Type:    apperr.Internal,
			Message: "Błąd podczas pobierania danych umowy.",
			Err:     err,
		}
	}
	if agreement == nil {
		return nil, apperr.ErrNotFound
	}

	// 2. Pobieranie zaszyfrowanego pliku z magazynu S3/MinIO
	encryptedPdfBytes, err := s.storage.Download(ctx, agreement.S3Key)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "STORAGE_DOWNLOAD_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd pobierania pliku umowy z magazynu.",
			Err:     err,
		}
	}

	// 3. Odszyfrowanie pliku w locie (Envelope Encryption)
	encPayload := envelope.EncryptedPayload{
		EncryptedData: encryptedPdfBytes,
		EncryptedDEK:  agreement.EncryptedDEK,
		KeyVersion:    agreement.KeyVersion,
	}

	decryptedPdf, err := s.cryptor.Unseal(ctx, s.agreementsKeyAlias, encPayload)
	if err != nil {
		return nil, &apperr.AppError{
			Code:    "DECRYPTION_FAILED",
			Type:    apperr.Internal,
			Message: "Błąd odszyfrowywania dokumentu umowy.",
			Err:     err,
		}
	}

	return decryptedPdf, nil
}
