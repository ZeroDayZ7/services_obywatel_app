// internal/di/container.go
package di

import (
	"gorm.io/gorm"
)

// Container przechowuje wszystkie zależności serwisów i handlerów
type Container struct {
}

// NewContainer tworzy nowy kontener z wszystkimi zależnościami
func NewContainer(db *gorm.DB) *Container {
	// repozytorium użytkowników

	// handlery
	return &Container{}
}
