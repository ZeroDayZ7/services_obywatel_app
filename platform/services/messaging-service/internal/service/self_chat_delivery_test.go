package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
)

type fakeSelfChatRepo struct {
	repository.MessagingRepository
	conversation *model.Conversation
}

func (r *fakeSelfChatRepo) GetConversationByID(_ context.Context, _ uuid.UUID, conversationID uuid.UUID) (*model.Conversation, error) {
	if r.conversation != nil && r.conversation.ID == conversationID {
		return r.conversation, nil
	}
	return nil, nil
}

func TestMessagingServicePublishesSelfConversationMessagesToTheOwner(t *testing.T) {
	userID := uuid.New()
	convID := uuid.New()
	publisher := &capturePublisher{}
	service := &messagingService{
		repo: &fakeSelfChatRepo{conversation: &model.Conversation{
			ID:   convID,
			Type: model.ConversationTypeDirect,
			Members: []model.ConversationMember{{
				UserID: userID,
				Role:   "admin",
			}},
		}},
		logger:    shared.InitLogger("development", false),
		publisher: publisher,
	}

	msg := &model.Message{
		ID:             uuid.New(),
		ConversationID: convID,
		SenderID:       userID,
		SenderDeviceID: "device-1",
		Version:        1,
		Type:           model.MessageTypeText,
	}

	service.publishMessageEvent(context.Background(), msg)

	if publisher.event == nil {
		t.Fatal("expected publisher to receive an event")
	}

	targetUsers, ok := publisher.event["targetUserIds"].([]string)
	if !ok {
		t.Fatalf("expected targetUserIds to be []string, got %T", publisher.event["targetUserIds"])
	}
	if len(targetUsers) != 1 || targetUsers[0] != userID.String() {
		t.Fatalf("expected self user %s in targetUserIds, got %v", userID.String(), targetUsers)
	}
}
