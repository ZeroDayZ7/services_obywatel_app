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
	s.logger.Info("[SERVICE-CONTACTS-01] GetContacts: starting retrieval",
		"user_id", userID.String(),
	)

	contacts, err := s.repo.GetContactsByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("[SERVICE-CONTACTS-02] GetContacts: repository failed to fetch contacts",
			"user_id", userID.String(),
			"error", err,
		)
		return nil, err
	}

	s.logger.Debug("[SERVICE-CONTACTS-03] GetContacts: raw contacts fetched successfully",
		"user_id", userID.String(),
		"count", len(contacts),
	)

	for i := range contacts {
		if contacts[i].OwnerID == userID {
			contacts[i].Direction = model.ContactDirectionOutgoing
			continue
		}
		if contacts[i].ContactID == userID {
			contacts[i].Direction = model.ContactDirectionIncoming
		}
	}

	s.logger.Info("[SERVICE-CONTACTS-04] GetContacts: directions normalized successfully",
		"user_id", userID.String(),
		"total_processed", len(contacts),
	)

	return contacts, nil
}

func (s *contactsService) SendRequest(ctx context.Context, ownerID, targetID uuid.UUID) (*model.Contact, error) {
	s.logger.Info("[SERVICE-CONTACTS-05] SendRequest: initiating contact request creation",
		"owner_id", ownerID.String(),
		"target_id", targetID.String(),
	)

	// 1. Sprawdzamy czy relacja już istnieje w dowolnym kierunku
	existing, err := s.repo.GetContactByOwnerAndTarget(ctx, ownerID, targetID)
	if err != nil {
		s.logger.Error("[SERVICE-CONTACTS-06] SendRequest: failed to check existing relation",
			"owner_id", ownerID.String(),
			"target_id", targetID.String(),
			"error", err,
		)
		return nil, err
	}
	if existing != nil {
		s.logger.Warn("[SERVICE-CONTACTS-07] SendRequest: contact relation already exists",
			"owner_id", ownerID.String(),
			"target_id", targetID.String(),
			"existing_contact_id", existing.ID.String(),
		)
		return nil, ErrContactAlreadyExists
	}

	// 2. Tworzymy nowe zaproszenie ze stanem 'pending'
	newContact := &model.Contact{
		OwnerID:   ownerID,
		ContactID: targetID,
		Status:    model.ContactStatusPending,
		Direction: model.ContactDirectionOutgoing,
		Version:   1,
	}

	if err := s.repo.CreateContact(ctx, newContact); err != nil {
		s.logger.Error("[SERVICE-CONTACTS-08] SendRequest: failed to persist new contact request",
			"owner_id", ownerID.String(),
			"target_id", targetID.String(),
			"error", err,
		)
		return nil, err
	}

	s.logger.Info("[SERVICE-CONTACTS-09] SendRequest: contact request created successfully",
		"owner_id", ownerID.String(),
		"target_id", targetID.String(),
		"contact_id", newContact.ID.String(),
	)

	return newContact, nil
}

func (s *contactsService) RespondToRequest(ctx context.Context, currentUserID, contactID uuid.UUID, accept bool) error {
	s.logger.Info("[SERVICE-CONTACTS-10] RespondToRequest: processing response for contact request",
		"current_user_id", currentUserID.String(),
		"contact_id", contactID.String(),
		"accept", accept,
	)

	// 1. Pobieramy zaproszenie z bazy
	contact, err := s.repo.GetContactByID(ctx, contactID)
	if err != nil {
		s.logger.Error("[SERVICE-CONTACTS-11] RespondToRequest: failed to fetch contact by id",
			"current_user_id", currentUserID.String(),
			"contact_id", contactID.String(),
			"error", err,
		)
		return err
	}
	if contact == nil {
		s.logger.Warn("[SERVICE-CONTACTS-12] RespondToRequest: contact request not found",
			"current_user_id", currentUserID.String(),
			"contact_id", contactID.String(),
		)
		return ErrContactNotFound
	}

	// 2. Tylko adresat zaproszenia (ContactID) może na nie odpowiedzieć
	if contact.ContactID != currentUserID {
		s.logger.Warn("[SERVICE-CONTACTS-13] RespondToRequest: unauthorized response attempt",
			"current_user_id", currentUserID.String(),
			"contact_id", contactID.String(),
			"expected_recipient_id", contact.ContactID.String(),
		)
		return ErrUnauthorizedAction
	}

	// 3. Obsługa odrzucenia
	if !accept {
		s.logger.Info("[SERVICE-CONTACTS-14] RespondToRequest: rejecting contact request (blocking)",
			"current_user_id", currentUserID.String(),
			"contact_id", contactID.String(),
		)
		if err := s.repo.UpdateContactStatus(ctx, contactID, model.ContactStatusBlocked); err != nil {
			s.logger.Error("[SERVICE-CONTACTS-15] RespondToRequest: failed to update status to blocked",
				"contact_id", contactID.String(),
				"error", err,
			)
			return err
		}
		s.logger.Info("[SERVICE-CONTACTS-16] RespondToRequest: contact request blocked successfully",
			"contact_id", contactID.String(),
		)
		return nil
	}

	// 4. Obsługa akceptacji - zmiana stanu u nadawcy
	s.logger.Info("[SERVICE-CONTACTS-17] RespondToRequest: accepting contact request",
		"current_user_id", currentUserID.String(),
		"contact_id", contactID.String(),
	)
	if err := s.repo.UpdateContactStatus(ctx, contactID, model.ContactStatusAccepted); err != nil {
		s.logger.Error("[SERVICE-CONTACTS-18] RespondToRequest: failed to update status to accepted",
			"contact_id", contactID.String(),
			"error", err,
		)
		return err
	}

	// 5. Utworzenie/zaktualizowanie relacji u odbiorcy (staje się symetryczna)
	s.logger.Info("[SERVICE-CONTACTS-19] RespondToRequest: creating symmetric contact relation",
		"current_user_id", currentUserID.String(),
		"owner_id", contact.OwnerID.String(),
	)
	if err := s.repo.CreateSymmetricContact(ctx, currentUserID, contact.OwnerID, model.ContactStatusAccepted); err != nil {
		s.logger.Error("[SERVICE-CONTACTS-20] RespondToRequest: failed to create symmetric contact",
			"current_user_id", currentUserID.String(),
			"owner_id", contact.OwnerID.String(),
			"error", err,
		)
		return err
	}

	s.logger.Info("[SERVICE-CONTACTS-21] RespondToRequest: contact request successfully accepted and symmetrized",
		"current_user_id", currentUserID.String(),
		"contact_id", contactID.String(),
	)

	return nil
}
