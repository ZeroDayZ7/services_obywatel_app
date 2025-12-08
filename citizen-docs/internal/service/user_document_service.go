package service

import (
	"github.com/zerodayz7/http-server/internal/model"
	"github.com/zerodayz7/http-server/internal/repository"
)

type UserDocumentService struct {
	repo repository.UserDocumentRepo
}

func NewUserDocumentService(repo repository.UserDocumentRepo) *UserDocumentService {
	return &UserDocumentService{repo: repo}
}

// Tworzy dokument
func (s *UserDocumentService) CreateDocument(doc *model.UserDocument) error {
	return s.repo.Create(doc)
}

// Pobiera dokumenty dla konkretnego userID
func (s *UserDocumentService) GetDocumentsByUserID(userID uint) ([]model.UserDocument, error) {
	return s.repo.GetByUserID(userID)
}
