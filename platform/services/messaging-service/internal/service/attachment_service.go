package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/storage"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
)

const defaultAttachmentUploadTTL = 15 * time.Minute

var maxAttachmentSizeBytes int64 = 25 * 1024 * 1024

type AttachmentService interface {
	PrepareUpload(ctx context.Context, userID uuid.UUID, req model.AttachmentUploadRequest) (*model.AttachmentUploadResponse, error)
	FinalizeUpload(ctx context.Context, userID uuid.UUID, req model.AttachmentFinalizeRequest) (*model.Attachment, error)
	GetAttachment(ctx context.Context, userID uuid.UUID, attachmentID uuid.UUID) (*model.Attachment, error)
	DeleteAttachment(ctx context.Context, userID uuid.UUID, attachmentID uuid.UUID) error
}

type attachmentService struct {
	repo    repository.MessagingRepository
	storage storage.StorageClient
	logger  *shared.Logger
	maxSize int64
}

func NewAttachmentService(repo repository.MessagingRepository, storageClient storage.StorageClient, logger *shared.Logger, maxSize int64) AttachmentService {
	if maxSize <= 0 {
		maxSize = maxAttachmentSizeBytes
	}
	return &attachmentService{
		repo:    repo,
		storage: storageClient,
		logger:  logger,
		maxSize: maxSize,
	}
}

func (s *attachmentService) PrepareUpload(ctx context.Context, userID uuid.UUID, req model.AttachmentUploadRequest) (*model.AttachmentUploadResponse, error) {
	if userID == uuid.Nil {
		return nil, apperr.ErrUnauthorized
	}
	if strings.TrimSpace(req.FileName) == "" {
		return nil, apperr.ErrValidationFailed.WithMeta("detail", "fileName is required")
	}
	if req.Size <= 0 {
		return nil, apperr.ErrValidationFailed.WithMeta("detail", "size must be greater than zero")
	}
	if s.maxSize > 0 && req.Size > s.maxSize {
		return nil, apperr.ErrValidationFailed.WithMeta("detail", fmt.Sprintf("attachment exceeds max size of %d bytes", s.maxSize))
	}
	if strings.TrimSpace(req.ContentType) == "" {
		req.ContentType = "application/octet-stream"
	}

	attachmentID := uuid.New()
	fileKey := fmt.Sprintf("attachments/%s/%s/%s", userID.String(), attachmentID.String(), sanitizeAttachmentFileName(req.FileName))
	attachment := &model.Attachment{
		ID:             attachmentID,
		UserID:         userID,
		ConversationID: req.ConversationID,
		FileKey:        fileKey,
		OriginalName:   req.FileName,
		ContentType:    req.ContentType,
		Size:           req.Size,
		Status:         model.AttachmentStatusPending,
	}
	if err := s.repo.CreateAttachment(ctx, attachment); err != nil {
		return nil, err
	}

	if s.storage == nil {
		return nil, apperr.ErrInternal.WithMeta("detail", "storage backend is not configured")
	}
	uploadURL, err := s.storage.GetPresignedUploadURL(ctx, attachment.FileKey, defaultAttachmentUploadTTL)
	if err != nil {
		return nil, apperr.ErrInternal.WithMeta("detail", "failed to create upload url")
	}

	return &model.AttachmentUploadResponse{
		Attachment: attachment,
		UploadURL:  uploadURL,
		ExpiresAt:  time.Now().Add(defaultAttachmentUploadTTL),
	}, nil
}

func (s *attachmentService) FinalizeUpload(ctx context.Context, userID uuid.UUID, req model.AttachmentFinalizeRequest) (*model.Attachment, error) {
	if userID == uuid.Nil {
		return nil, apperr.ErrUnauthorized
	}
	if req.AttachmentID == uuid.Nil {
		return nil, apperr.ErrValidationFailed.WithMeta("detail", "attachmentId is required")
	}

	attachment, err := s.repo.GetAttachmentByID(ctx, req.AttachmentID)
	if err != nil {
		return nil, err
	}
	if attachment == nil {
		return nil, apperr.ErrNotFound
	}
	if attachment.UserID != userID {
		return nil, apperr.ErrUnauthorized.WithMeta("detail", "attachment does not belong to this user")
	}
	if attachment.Status == model.AttachmentStatusDeleted {
		return nil, apperr.ErrNotFound
	}
	if req.Size > 0 && attachment.Size != req.Size {
		return nil, apperr.ErrValidationFailed.WithMeta("detail", "attachment size mismatch")
	}

	if s.storage != nil {
		if _, err := s.storage.HeadObject(ctx, attachment.FileKey); err != nil {
			return nil, apperr.ErrValidationFailed.WithMeta("detail", "attachment file has not been uploaded yet")
		}
		payload, err := s.storage.Download(ctx, attachment.FileKey)
		if err != nil {
			return nil, apperr.ErrValidationFailed.WithMeta("detail", "failed to read uploaded attachment")
		}
		sum := sha256.Sum256(payload)
		realHash := hex.EncodeToString(sum[:])
		if strings.TrimSpace(req.Sha256) != "" && !strings.EqualFold(realHash, strings.TrimSpace(req.Sha256)) {
			return nil, apperr.ErrValidationFailed.WithMeta("detail", "sha256 mismatch")
		}
		attachment.Sha256 = realHash
	}
	if strings.TrimSpace(req.Sha256) != "" && attachment.Sha256 == "" {
		attachment.Sha256 = strings.TrimSpace(req.Sha256)
	}
	attachment.Status = model.AttachmentStatusUploaded
	if err := s.repo.UpdateAttachment(ctx, attachment); err != nil {
		return nil, err
	}
	return attachment, nil
}

func (s *attachmentService) GetAttachment(ctx context.Context, userID uuid.UUID, attachmentID uuid.UUID) (*model.Attachment, error) {
	if userID == uuid.Nil {
		return nil, apperr.ErrUnauthorized
	}
	attachment, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return nil, err
	}
	if attachment == nil {
		return nil, apperr.ErrNotFound
	}
	if attachment.UserID != userID {
		return nil, apperr.ErrUnauthorized.WithMeta("detail", "attachment does not belong to this user")
	}
	return attachment, nil
}

func (s *attachmentService) DeleteAttachment(ctx context.Context, userID uuid.UUID, attachmentID uuid.UUID) error {
	if userID == uuid.Nil {
		return apperr.ErrUnauthorized
	}
	attachment, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return err
	}
	if attachment == nil {
		return apperr.ErrNotFound
	}
	if attachment.UserID != userID {
		return apperr.ErrUnauthorized.WithMeta("detail", "attachment does not belong to this user")
	}
	if s.storage != nil {
		if err := s.storage.Delete(ctx, attachment.FileKey); err != nil && !strings.Contains(err.Error(), "not found") {
			return apperr.ErrInternal.WithMeta("detail", "failed to delete attachment from storage")
		}
	}
	attachment.Status = model.AttachmentStatusDeleted
	if err := s.repo.UpdateAttachment(ctx, attachment); err != nil {
		return err
	}
	return nil
}

func sanitizeAttachmentFileName(fileName string) string {
	base := filepath.Base(strings.TrimSpace(fileName))
	if base == "." || base == "/" || base == "" {
		base = "attachment.bin"
	}
	return base
}
