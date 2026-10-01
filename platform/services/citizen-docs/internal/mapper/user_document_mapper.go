package mapper

import (
	"encoding/json"
	"time"

	"github.com/zerodayz7/platform/services/citizen-docs/internal/dto"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
)

func ToDocumentResponse(doc model.CitizenDocument) dto.DocumentResponse {
	response := dto.DocumentResponse{
		ID:             doc.ID.String(),
		UserID:         doc.UserID.String(),
		DocumentType:   doc.DocumentType,
		DocumentNumber: doc.DocumentNumber,
		Status:         string(doc.Status),
		Metadata:       json.RawMessage(doc.Metadata),
		CreatedAt:      doc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      doc.UpdatedAt.Format(time.RFC3339),
	}
	if doc.IssuedAt != nil {
		response.IssuedAt = doc.IssuedAt.Format(time.RFC3339)
	}
	if doc.ExpiresAt != nil {
		response.ExpiresAt = doc.ExpiresAt.Format(time.RFC3339)
	}
	return response
}

func ToDocumentResponses(docs []model.CitizenDocument) []dto.DocumentResponse {
	if len(docs) == 0 {
		return []dto.DocumentResponse{}
	}

	resp := make([]dto.DocumentResponse, 0, len(docs))
	for _, doc := range docs {
		resp = append(resp, ToDocumentResponse(doc))
	}
	return resp
}

func ToUserDocumentResponse(doc model.CitizenDocument) dto.UserDocumentResponse {
	return ToDocumentResponse(doc)
}

func ToUserDocumentResponses(docs []model.CitizenDocument) []dto.UserDocumentResponse {
	response := make([]dto.UserDocumentResponse, 0, len(docs))
	for _, doc := range docs {
		response = append(response, ToDocumentResponse(doc))
	}
	return response
}
