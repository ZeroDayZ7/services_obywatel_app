package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
)

var (
	ErrContactAlreadyExists = errors.New("contact relation already exists")
	ErrContactNotFound      = errors.New("contact request not found")
	ErrUnauthorizedAction   = errors.New("unauthorized to respond to this request")
)

type ContactsService interface {
	GetContacts(ctx context.Context, userID uuid.UUID) ([]model.Contact, error)
	SendRequest(ctx context.Context, ownerID, targetID uuid.UUID) (*model.Contact, error)
	RespondToRequest(ctx context.Context, currentUserID, contactID uuid.UUID, accept bool) error
}

type contactsService struct {
	repo   repository.ContactsRepository
	logger *shared.Logger
}

func NewContactsService(repo repository.ContactsRepository, logger *shared.Logger) ContactsService {
	return &contactsService{
		repo:   repo,
		logger: logger,
	}
}

func (s *contactsService) GetContacts(ctx context.Context, userID uuid.UUID) ([]model.Contact, error) {
	return s.repo.GetContactsByUserID(ctx, userID)
}

func (s *contactsService) SendRequest(ctx context.Context, ownerID, targetID uuid.UUID) (*model.Contact, error) {
	// 1. Sprawdzamy czy relacja już istnieje
	existing, err := s.repo.GetContactByOwnerAndTarget(ctx, ownerID, targetID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrContactAlreadyExists
	}

	// 2. Tworzymy nowe zaproszenie ze stanem 'pending'
	newContact := &model.Contact{
		OwnerID:   ownerID,
		ContactID: targetID,
		Status:    model.ContactStatusPending,
		Version:   1,
	}

	if err := s.repo.CreateContact(ctx, newContact); err != nil {
		s.logger.ErrorObj("Failed to create contact request", map[string]any{
			"owner_id":  ownerID,
			"target_id": targetID,
			"error":     err.Error(),
		})
		return nil, err
	}

	return newContact, nil
}

func (s *contactsService) RespondToRequest(ctx context.Context, currentUserID, contactID uuid.UUID, accept bool) error {
	// 1. Pobieramy zaproszenie z bazy
	contact, err := s.repo.GetContactByID(ctx, contactID)
	if err != nil {
		return err
	}
	if contact == nil {
		return ErrContactNotFound
	}

	// 2. Tylko adresat zaproszenia (ContactID) może na nie odpowiedzieć
	if contact.ContactID != currentUserID {
		return ErrUnauthorizedAction
	}

	// 3. Obsługa odrzucenia
	if !accept {
		return s.repo.UpdateContactStatus(ctx, contactID, model.ContactStatusBlocked)
	}

	// 4. Obsługa akceptacji - zmiana stanu u nadawcy
	if err := s.repo.UpdateContactStatus(ctx, contactID, model.ContactStatusAccepted); err != nil {
		return err
	}

	// 5. Utworzenie/zaktualizowanie relacji u odbiorcy (staje się symetryczna)
	return s.repo.CreateSymmetricContact(ctx, currentUserID, contact.OwnerID, model.ContactStatusAccepted)
}
