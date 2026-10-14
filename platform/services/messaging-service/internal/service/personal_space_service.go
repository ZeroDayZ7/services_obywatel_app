package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
)

type PersonalSpaceService interface {
	GetOrCreateSpace(ctx context.Context, ownerID uuid.UUID) (*model.PersonalSpace, error)
	ListItems(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.PersonalSpaceItem, error)
	CreateItem(ctx context.Context, ownerID uuid.UUID, req model.CreatePersonalSpaceItemRequest) (*model.PersonalSpaceItem, error)
	GetItem(ctx context.Context, ownerID, itemID uuid.UUID) (*model.PersonalSpaceItem, error)
	UpdateItem(ctx context.Context, ownerID uuid.UUID, req model.UpdatePersonalSpaceItemRequest) (*model.PersonalSpaceItem, error)
	DeleteItem(ctx context.Context, ownerID, itemID uuid.UUID) error
}

type personalSpaceService struct {
	repo repository.PersonalSpaceRepository
}

func NewPersonalSpaceService(repo repository.PersonalSpaceRepository) PersonalSpaceService {
	return &personalSpaceService{repo: repo}
}

func (s *personalSpaceService) GetOrCreateSpace(ctx context.Context, ownerID uuid.UUID) (*model.PersonalSpace, error) {
	if ownerID == uuid.Nil {
		return nil, errors.New("owner id is required")
	}
	return s.repo.GetOrCreateSpace(ctx, ownerID)
}

func (s *personalSpaceService) ListItems(ctx context.Context, ownerID uuid.UUID, sinceVersion uint64) ([]model.PersonalSpaceItem, error) {
	if ownerID == uuid.Nil {
		return nil, errors.New("owner id is required")
	}
	return s.repo.ListItems(ctx, ownerID, sinceVersion)
}

func (s *personalSpaceService) CreateItem(ctx context.Context, ownerID uuid.UUID, req model.CreatePersonalSpaceItemRequest) (*model.PersonalSpaceItem, error) {
	if ownerID == uuid.Nil {
		return nil, errors.New("owner id is required")
	}
	if len(req.EncryptedPayload) == 0 {
		return nil, errors.New("encrypted payload is required")
	}
	if req.ContentType == "" {
		req.ContentType = model.PersonalSpaceItemTypeText
	}
	space, err := s.repo.GetOrCreateSpace(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if req.SpaceID != nil && *req.SpaceID != uuid.Nil && *req.SpaceID != space.ID {
		return nil, fmt.Errorf("space id mismatch")
	}
	item := &model.PersonalSpaceItem{
		ID:               uuid.New(),
		SpaceID:          space.ID,
		OwnerID:          ownerID,
		DeviceID:         strings.TrimSpace(req.DeviceID),
		Title:            strings.TrimSpace(req.Title),
		ContentType:      req.ContentType,
		EncryptedPayload: req.EncryptedPayload,
		Status:           model.PersonalSpaceItemStatusActive,
		SyncVersion:      space.SyncVersion + 1,
		Source:           "device",
	}
	if item.DeviceID == "" {
		item.DeviceID = "device-local"
	}
	if err := s.repo.CreateItem(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *personalSpaceService) GetItem(ctx context.Context, ownerID, itemID uuid.UUID) (*model.PersonalSpaceItem, error) {
	if ownerID == uuid.Nil {
		return nil, errors.New("owner id is required")
	}
	if itemID == uuid.Nil {
		return nil, errors.New("item id is required")
	}
	return s.repo.GetItemByID(ctx, ownerID, itemID)
}

func (s *personalSpaceService) UpdateItem(ctx context.Context, ownerID uuid.UUID, req model.UpdatePersonalSpaceItemRequest) (*model.PersonalSpaceItem, error) {
	if ownerID == uuid.Nil {
		return nil, errors.New("owner id is required")
	}
	if req.ItemID == uuid.Nil {
		return nil, errors.New("item id is required")
	}
	item, err := s.repo.GetItemByID(ctx, ownerID, req.ItemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, errors.New("item not found")
	}
	if strings.TrimSpace(req.Title) != "" {
		item.Title = strings.TrimSpace(req.Title)
	}
	if req.ContentType != "" {
		item.ContentType = req.ContentType
	}
	if len(req.EncryptedPayload) > 0 {
		item.EncryptedPayload = req.EncryptedPayload
	}
	if strings.TrimSpace(req.DeviceID) != "" {
		item.DeviceID = req.DeviceID
	}
	if req.SyncVersion > 0 {
		item.SyncVersion = req.SyncVersion
	}
	item.SyncVersion++
	if err := s.repo.UpdateItem(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *personalSpaceService) DeleteItem(ctx context.Context, ownerID, itemID uuid.UUID) error {
	if ownerID == uuid.Nil {
		return errors.New("owner id is required")
	}
	if itemID == uuid.Nil {
		return errors.New("item id is required")
	}
	return s.repo.DeleteItem(ctx, ownerID, itemID)
}
