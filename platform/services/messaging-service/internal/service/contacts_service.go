package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	pub "github.com/zerodayz7/platform/services/messaging-service/internal/publisher"
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
	repo      repository.ContactsRepository
	logger    *shared.Logger
	publisher pub.EventPublisher
}

func NewContactsService(repo repository.ContactsRepository, logger *shared.Logger, publisher pub.EventPublisher) ContactsService {
	return &contactsService{
		repo:      repo,
		logger:    logger,
		publisher: publisher,
	}
}

func (s *contactsService) GetContacts(ctx context.Context, userID uuid.UUID) ([]model.Contact, error) {
	contacts, err := s.repo.GetContactsByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to fetch contacts", "user_id", userID.String(), "error", err)
		return nil, err
	}

	for i := range contacts {
		if contacts[i].OwnerID == userID {
			contacts[i].Direction = model.ContactDirectionOutgoing
			continue
		}
		if contacts[i].ContactID == userID {
			contacts[i].Direction = model.ContactDirectionIncoming
		}
	}

	return contacts, nil
}

func (s *contactsService) SendRequest(ctx context.Context, ownerID, targetID uuid.UUID) (*model.Contact, error) {
	existing, err := s.repo.GetContactByOwnerAndTarget(ctx, ownerID, targetID)
	if err != nil {
		s.logger.Error("failed to check existing contact relation", "owner_id", ownerID.String(), "target_id", targetID.String(), "error", err)
		return nil, err
	}
	if existing != nil {
		s.logger.Warn("contact relation already exists", "owner_id", ownerID.String(), "target_id", targetID.String())
		return nil, ErrContactAlreadyExists
	}

	newContact := &model.Contact{
		OwnerID:   ownerID,
		ContactID: targetID,
		Status:    model.ContactStatusPending,
		Direction: model.ContactDirectionOutgoing,
		Version:   1,
	}

	if err := s.repo.CreateContact(ctx, newContact); err != nil {
		s.logger.Error("failed to persist contact request", "owner_id", ownerID.String(), "target_id", targetID.String(), "error", err)
		return nil, err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, pub.NewEvent([]uuid.UUID{ownerID, targetID}, map[string]any{
			"type":           "contact_updated",
			"event_id":       uuid.NewString(),
			"contact_id":     newContact.ID.String(),
			"owner_user_id":  ownerID.String(),
			"target_user_id": targetID.String(),
			"status":         string(newContact.Status),
			"direction":      string(newContact.Direction),
			"timestamp":      time.Now().UnixMilli(),
		})); err != nil {
			s.logger.Warn("contact event publication failed after persisting contact request",
				"owner_id", ownerID.String(),
				"target_id", targetID.String(),
				"contact_id", newContact.ID.String(),
				"error", err,
			)
		}
	}

	return newContact, nil
}

func (s *contactsService) RespondToRequest(ctx context.Context, currentUserID, contactID uuid.UUID, accept bool) error {
	contact, err := s.repo.GetContactByID(ctx, contactID)
	if err != nil {
		s.logger.Error("failed to fetch contact for response", "current_user_id", currentUserID.String(), "contact_id", contactID.String(), "error", err)
		return err
	}
	if contact == nil {
		s.logger.Warn("contact request not found", "current_user_id", currentUserID.String(), "contact_id", contactID.String())
		return ErrContactNotFound
	}

	if contact.ContactID != currentUserID {
		s.logger.Warn("unauthorized contact response attempt", "current_user_id", currentUserID.String(), "contact_id", contactID.String(), "expected_recipient_id", contact.ContactID.String())
		return ErrUnauthorizedAction
	}

	if !accept {
		if err := s.repo.UpdateContactStatus(ctx, contactID, model.ContactStatusBlocked); err != nil {
			s.logger.Error("failed to block contact request", "contact_id", contactID.String(), "error", err)
			return err
		}
		return nil
	}

	if err := s.repo.UpdateContactStatus(ctx, contactID, model.ContactStatusAccepted); err != nil {
		s.logger.Error("failed to accept contact request", "contact_id", contactID.String(), "error", err)
		return err
	}

	if err := s.repo.CreateSymmetricContact(ctx, currentUserID, contact.OwnerID, model.ContactStatusAccepted); err != nil {
		s.logger.Error("failed to create symmetric contact relation", "current_user_id", currentUserID.String(), "owner_id", contact.OwnerID.String(), "error", err)
		return err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, pub.NewEvent([]uuid.UUID{contact.OwnerID, currentUserID}, map[string]any{
			"type":           "contact_updated",
			"event_id":       uuid.NewString(),
			"contact_id":     contactID.String(),
			"owner_user_id":  contact.OwnerID.String(),
			"target_user_id": currentUserID.String(),
			"status":         model.ContactStatusAccepted,
			"timestamp":      time.Now().UnixMilli(),
		})); err != nil {
			s.logger.Warn("contact event publication failed after accepting contact request",
				"current_user_id", currentUserID.String(),
				"contact_id", contactID.String(),
				"owner_id", contact.OwnerID.String(),
				"error", err,
			)
		}
	}

	return nil
}
