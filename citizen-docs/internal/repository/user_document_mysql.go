package repository

import (
	"github.com/zerodayz7/http-server/internal/model"
	"gorm.io/gorm"
)

// UserDocumentRepositoryGORM implements UserDocumentRepo
type UserDocumentRepositoryGORM struct {
	db *gorm.DB
}

// zwraca interfejs, nie konkretną strukturę
func NewUserDocumentRepositoryGORM(db *gorm.DB) UserDocumentRepo {
	return &UserDocumentRepositoryGORM{db: db}
}

func (r *UserDocumentRepositoryGORM) Create(doc *model.UserDocument) error {
	return r.db.Create(doc).Error
}

func (r *UserDocumentRepositoryGORM) GetByUserID(userID uint) ([]model.UserDocument, error) {
	var docs []model.UserDocument
	err := r.db.Where("user_id = ?", userID).Find(&docs).Error
	return docs, err
}
