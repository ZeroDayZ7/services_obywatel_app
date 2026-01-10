// platform/services/citizen-docs/internal/service/user_document_service.go

package service

import (
	"context"
	"encoding/json"

	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/repository"
)

type UserDocumentService struct {
	repo   repository.UserDocumentRepo
	cfg    *viper.Config
	logger *shared.Logger
}

func NewUserDocumentService(repo repository.UserDocumentRepo, cfg *viper.Config, logger *shared.Logger) *UserDocumentService {
	return &UserDocumentService{
		repo:   repo,
		cfg:    cfg,
		logger: logger,
	}
}

// CreateDocument dopasowany do wywołania z handlera
func (s *UserDocumentService) CreateDocument(
	ctx context.Context,
	meta *model.DocumentMeta,
	front []byte,
	back []byte,
	profileID uint,
	docType model.DocumentType,
) error {
	encryptionKey := []byte(s.cfg.Internal.EncryptionKey)

	// Szyfrowanie metadanych (JSON)
	metaBytes, _ := json.Marshal(meta)
	encMeta, err := shared.Encrypt(metaBytes, encryptionKey)
	if err != nil {
		return err
	}

	// Szyfrowanie plików binarnych
	encFront, _ := shared.Encrypt(front, encryptionKey)
	encBack, _ := shared.Encrypt(back, encryptionKey)

	doc := &model.UserDocument{
		ProfileID:      profileID,
		Type:           docType,
		EncryptedMeta:  encMeta,
		EncryptedFront: encFront,
		EncryptedBack:  encBack,
		Status:         model.DocumentStatusActive,
	}

	return s.repo.Create(ctx, doc)
}

// Ta metoda rozwiązuje błąd 'GetDocumentsByUserID undefined'
// (Zmieniłem nazwę na GetDocumentsByProfileID, upewnij się że w handlerze wywołasz ją poprawnie)
func (s *UserDocumentService) GetDocumentsByProfileID(ctx context.Context, profileID uint) ([]model.UserDocument, error) {
	return s.repo.GetByProfileID(ctx, profileID)
}
