package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/crypto"
	"github.com/zerodayz7/platform/pkg/envelope"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/config"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/repository"
	"gorm.io/datatypes"
)

func resolveDocumentKeyAlias(documentType string, fallback string) string {
	switch strings.ToUpper(strings.TrimSpace(documentType)) {
	case "ID_CARD", "ID-CARD":
		return "docs-id-cards"
	case "DRIVERS_LICENSE", "DRIVER_LICENSE", "DRIVERS-LICENSE":
		return "docs-driver-license"
	case "PASSPORT":
		return "docs-passport"
	default:
		return fallback
	}
}

type userDocumentService struct {
	docRepo                  repository.UserDocumentRepo
	cfg                      *config.Config
	cryptor                  *envelope.EnvelopeCryptor
	hmacDocumentNumberSecret []byte
	metadataKeyAlias         string
}

func NewUserDocumentService(
	docRepo repository.UserDocumentRepo,
	cfg *config.Config,
	cryptor *envelope.EnvelopeCryptor,
	hmacDocumentNumberSecret []byte,
	metadataKeyAlias string,
) UserDocumentService {
	return &userDocumentService{
		docRepo:                  docRepo,
		cfg:                      cfg,
		cryptor:                  cryptor,
		hmacDocumentNumberSecret: hmacDocumentNumberSecret,
		metadataKeyAlias:         metadataKeyAlias,
	}
}

func computeDocumentNumberHash(documentNumber string, secret []byte) string {
	return crypto.ComputeHMAC256Hex([]byte(documentNumber), secret)
}

func (s *userDocumentService) decryptDocumentMetadata(ctx context.Context, doc *model.CitizenDocument) (*model.CitizenDocument, error) {
	if doc == nil {
		return nil, nil
	}
	if s.cryptor == nil {
		return nil, fmt.Errorf("document cryptor is not configured")
	}
	if len(doc.EncryptedMetadata) == 0 && len(doc.EncryptedDEK) == 0 {
		if len(doc.Metadata) == 0 {
			doc.Metadata = datatypes.JSON(`{}`)
		}
		return doc, nil
	}

	keyAlias := resolveDocumentKeyAlias(doc.DocumentType, s.metadataKeyAlias)
	if keyAlias == "" {
		return nil, fmt.Errorf("document metadata alias is not configured")
	}
	plaintext, err := s.cryptor.OpenWithDataKey(ctx, keyAlias, doc.EncryptedMetadata, doc.EncryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt document metadata with KMS alias %s: %w", keyAlias, err)
	}
	if len(plaintext) == 0 {
		plaintext = []byte(`{}`)
	}
	doc.Metadata = datatypes.JSON(plaintext)
	return doc, nil
}

func (s *userDocumentService) CreateDocument(ctx context.Context, payload model.CreateDocumentPayload) (*model.CitizenDocument, error) {
	if payload.UserID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}
	if payload.DocumentType == "" {
		return nil, fmt.Errorf("document_type is required")
	}
	if payload.DocumentNumber == "" {
		payload.DocumentNumber = fmt.Sprintf("%s-%s", strings.ToUpper(payload.DocumentType), strings.ToLower(uuid.NewString()[:8]))
	}
	if payload.Status == "" {
		payload.Status = model.DocumentStatusPending
	}
	if payload.Metadata == nil {
		payload.Metadata = datatypes.JSON(`{}`)
	}
	if s.cryptor == nil {
		return nil, fmt.Errorf("document cryptor is not configured")
	}
	if len(s.hmacDocumentNumberSecret) == 0 {
		return nil, fmt.Errorf("document number HMAC secret is not configured")
	}

	metadataBytes := []byte(payload.Metadata)
	if len(metadataBytes) == 0 {
		metadataBytes = []byte(`{}`)
	}

	keyAlias := resolveDocumentKeyAlias(payload.DocumentType, s.metadataKeyAlias)
	if keyAlias == "" {
		return nil, fmt.Errorf("document metadata alias is not configured")
	}
	encryptedPayload, err := s.cryptor.SealWithDataKey(ctx, keyAlias, metadataBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt document metadata via KMS alias %s: %w", keyAlias, err)
	}

	doc := &model.CitizenDocument{
		UserID:             payload.UserID,
		DocumentType:       strings.ToUpper(payload.DocumentType),
		DocumentNumberHash: computeDocumentNumberHash(payload.DocumentNumber, s.hmacDocumentNumberSecret),
		Status:             payload.Status,
		Metadata:           payload.Metadata,
		EncryptedMetadata:  encryptedPayload.EncryptedData,
		EncryptedDEK:       encryptedPayload.EncryptedDEK,
		IssuedAt:           payload.IssuedAt,
		ExpiresAt:          payload.ExpiresAt,
	}

	if err := s.docRepo.CreateDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed to create citizen document: %w", err)
	}

	return s.decryptDocumentMetadata(ctx, doc)
}

func (s *userDocumentService) GetDocumentByID(ctx context.Context, id uuid.UUID) (*model.CitizenDocument, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("document id is required")
	}

	doc, err := s.docRepo.GetDocumentByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch document %s: %w", id, err)
	}
	if doc == nil {
		return nil, nil
	}
	return s.decryptDocumentMetadata(ctx, doc)
}

func (s *userDocumentService) GetDocumentByDocumentNumber(ctx context.Context, documentNumber string) (*model.CitizenDocument, error) {
	if documentNumber == "" {
		return nil, fmt.Errorf("document_number is required")
	}
	if len(s.hmacDocumentNumberSecret) == 0 {
		return nil, fmt.Errorf("document number HMAC secret is not configured")
	}

	doc, err := s.docRepo.GetDocumentByDocumentNumberHash(ctx, computeDocumentNumberHash(documentNumber, s.hmacDocumentNumberSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch document by document_number: %w", err)
	}
	if doc == nil {
		return nil, nil
	}
	return s.decryptDocumentMetadata(ctx, doc)
}

func (s *userDocumentService) GetDocumentsByUserID(ctx context.Context, userID uuid.UUID) ([]model.CitizenDocument, error) {
	log := shared.GetLogger()
	log.InfoMap("[userDocumentService.GetDocumentsByUserID] 1. Validating request", map[string]any{"user_id": userID.String()})

	if userID == uuid.Nil {
		log.Warn("[userDocumentService.GetDocumentsByUserID] 1.1. Missing user_id")
		return nil, fmt.Errorf("user_id is required")
	}

	log.InfoMap("[userDocumentService.GetDocumentsByUserID] 2. Querying document repository", map[string]any{"user_id": userID.String()})
	docs, err := s.docRepo.GetDocumentsByUserID(ctx, userID)
	if err != nil {
		log.ErrorMap("[userDocumentService.GetDocumentsByUserID] 3. Repository query failed", map[string]any{"user_id": userID.String(), "err": err.Error()})
		return nil, fmt.Errorf("failed to fetch documents for user %s: %w", userID, err)
	}
	if docs == nil {
		log.InfoMap("[userDocumentService.GetDocumentsByUserID] 4. Repository returned nil; normalizing to empty slice", map[string]any{"user_id": userID.String()})
		return make([]model.CitizenDocument, 0), nil
	}
	for i := range docs {
		if decrypted, err := s.decryptDocumentMetadata(ctx, &docs[i]); err != nil {
			return nil, err
		} else {
			docs[i] = *decrypted
		}
	}
	log.InfoMap("[userDocumentService.GetDocumentsByUserID] 5. Documents loaded successfully", map[string]any{"user_id": userID.String(), "count": len(docs)})
	return docs, nil
}

func (s *userDocumentService) UpdateDocumentStatus(ctx context.Context, id uuid.UUID, status model.DocumentStatus) (*model.CitizenDocument, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("document id is required")
	}
	if status == "" {
		return nil, fmt.Errorf("status is required")
	}

	doc, err := s.docRepo.UpdateDocumentStatus(ctx, id, status)
	if err != nil {
		return nil, fmt.Errorf("failed to update document %s status: %w", id, err)
	}
	return doc, nil
}

func (s *userDocumentService) GetDocumentPDF(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	doc, err := s.GetDocumentByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	return buildDocumentPDF(doc), "application/pdf", nil
}

func buildDocumentPDF(doc *model.CitizenDocument) []byte {
	header := fmt.Sprintf("Citizen Document\nType: %s\nUserID: %s\nStatus: %s",
		doc.DocumentType,
		doc.UserID.String(),
		doc.Status,
	)
	content := escapePDFText(header)
	stream := "BT\n/F1 12 Tf\n72 720 Td\n(" + content + ") Tj\nET"
	streamLen := len(stream)

	pdf := "%PDF-1.4\n"
	pdf += "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n"
	pdf += "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n"
	pdf += "3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n"
	pdf += "4 0 obj\n<< /Length " + strconv.Itoa(streamLen) + " >>\nstream\n" + stream + "\nendstream\nendobj\n"
	pdf += "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n"
	pdf += "xref\n0 6\n0000000000 65535 f \n"
	pdf += "trailer\n<< /Root 1 0 R /Size 6 >>\nstartxref\n0\n%%EOF"
	return []byte(pdf)
}

func escapePDFText(s string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)", "\r", "", "\n", " ")
	return replacer.Replace(s)
}
