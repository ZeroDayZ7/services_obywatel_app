package repository

import "github.com/zerodayz7/http-server/internal/model"

type UserDocumentRepo interface {
	Create(doc *model.UserDocument) error
	GetByUserID(userID uint) ([]model.UserDocument, error)
}
