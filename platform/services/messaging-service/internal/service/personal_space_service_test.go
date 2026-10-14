package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPersonalSpaceServiceCreatesPrivateSpaceAndProtectsOwnerIsolation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.PersonalSpace{}, &model.PersonalSpaceItem{}); err != nil {
		t.Fatalf("bootstrap db: %v", err)
	}

	userA := uuid.New()
	userB := uuid.New()
	repo := repository.NewPersonalSpaceRepository(db)
	service := NewPersonalSpaceService(repo)

	spaceA, err := service.GetOrCreateSpace(context.Background(), userA)
	if err != nil {
		t.Fatalf("create owner A space: %v", err)
	}
	if spaceA.OwnerID != userA {
		t.Fatalf("expected owner A, got %s", spaceA.OwnerID)
	}

	item, err := service.CreateItem(context.Background(), userA, model.CreatePersonalSpaceItemRequest{
		Title:            "Notatka A",
		ContentType:      model.PersonalSpaceItemTypeText,
		EncryptedPayload: []byte("cipher-a"),
		DeviceID:         "device-a",
	})
	if err != nil {
		t.Fatalf("create item for A: %v", err)
	}
	if item.OwnerID != userA {
		t.Fatalf("expected item owner A, got %s", item.OwnerID)
	}

	spaceB, err := service.GetOrCreateSpace(context.Background(), userB)
	if err != nil {
		t.Fatalf("create owner B space: %v", err)
	}
	if spaceA.ID == spaceB.ID {
		t.Fatalf("space IDs should differ per owner")
	}

	items, err := service.ListItems(context.Background(), userB, 0)
	if err != nil {
		t.Fatalf("list items for owner B: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("owner B should not see owner A's private items")
	}
}
