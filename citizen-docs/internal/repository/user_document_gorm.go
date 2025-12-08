package repository

import (
	"github.com/zerodayz7/http-server/internal/model"

	"gorm.io/gorm"
)

// UserDocumentRepository handles DB operations for user documents
type UserDocumentRepository struct {
	db *gorm.DB
}

// NewUserDocumentRepository creates a new repository instance
func NewUserDocumentRepository(db *gorm.DB) *UserDocumentRepository {
	return &UserDocumentRepository{db: db}
}

// Create saves a new user document
func (r *UserDocumentRepository) Create(doc *model.UserDocument) error {
	return r.db.Create(doc).Error
}

// GetByUserID retrieves all documents for a given user
func (r *UserDocumentRepository) GetByUserID(userID uint) ([]model.UserDocument, error) {
	var docs []model.UserDocument
	err := r.db.Where("user_id = ?", userID).Find(&docs).Error
	return docs, err
}
