package di

import (
	"github.com/zerodayz7/platform/services/citizen-docs/internal/repository"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/service"
	"gorm.io/gorm"
)

// Container przechowuje wszystkie zależności mikroserwisu
type Container struct {
	DB              *gorm.DB
	UserDocumentSvc *service.UserDocumentService
}

func NewContainer(db *gorm.DB) *Container {
	repo := repository.NewUserDocumentRepository(db)
	svc := service.NewUserDocumentService(repo)

	return &Container{
		DB:              db,
		UserDocumentSvc: svc,
	}
}
