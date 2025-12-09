package repository

import "../../../../../citizen-docs/internal/repository/github.com/zerodayz7/platform/services/citizen-docs/internal/model"

type UserDocumentRepo interface {
	Create(doc *model.UserDocument) error
	GetByUserID(userID uint) ([]model.UserDocument, error)
}
