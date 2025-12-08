package di

import (
	"github.com/zerodayz7/http-server/internal/repository"
	"gorm.io/gorm"
)

// Container przechowuje wszystkie zależności mikroserwisu
type Container struct {
	DB               *gorm.DB
	UserDocumentRepo *repository.UserDocumentRepository
	// dodaj inne serwisy i handlery tutaj
}

// NewContainer tworzy kontener z wszystkimi zależnościami
func NewContainer(db *gorm.DB) *Container {
	return &Container{
		DB:               db,
		UserDocumentRepo: repository.NewUserDocumentRepository(db),
	}
}
