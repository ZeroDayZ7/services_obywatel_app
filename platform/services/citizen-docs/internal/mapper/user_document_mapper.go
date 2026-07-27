package mapper

import (
	"encoding/base64"
	"time"

	"github.com/zerodayz7/platform/services/citizen-docs/internal/dto"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
)

func ToUserDocumentResponse(doc model.UserDocument) dto.UserDocumentResponse {
	response := dto.UserDocumentResponse{
		ID:            doc.ID.String(),
		Type:          string(doc.Type),
		Status:        string(doc.Status),
		EncryptedMeta: base64.StdEncoding.EncodeToString(doc.EncryptedMeta),
	}

	if doc.IssuedAt != nil {
		response.IssuedAt = doc.IssuedAt.Format(time.RFC3339)
	}

	if doc.ExpiresAt != nil {
		response.ExpiresAt = doc.ExpiresAt.Format(time.RFC3339)
	}

	return response
}

func ToUserDocumentResponses(docs []model.UserDocument) []dto.UserDocumentResponse {
	if len(docs) == 0 {
		return []dto.UserDocumentResponse{}
	}

	dtos := make([]dto.UserDocumentResponse, len(docs))
	for i, doc := range docs {
		dtos[i] = ToUserDocumentResponse(doc)
	}

	return dtos
}
