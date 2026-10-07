package mapper

import (
	"encoding/json"
	"time"

	"github.com/zerodayz7/platform/services/citizen-docs/internal/dto"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
)

func ToDocumentResponse(doc model.CitizenDocument, decryptedMetadata []byte) dto.DocumentResponse {
	response := dto.DocumentResponse{
		ID:           doc.ID.String(),
		UserID:       doc.UserID.String(),
		TypeCode:     doc.DocumentType,
		DocumentType: doc.DocumentType,
		Status:       string(doc.Status),
		DocumentHash: doc.DocumentNumberHash,
		Version:      doc.Version,
		CreatedAt:    doc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    doc.UpdatedAt.Format(time.RFC3339),
	}
	if doc.IssuedAt != nil {
		response.IssuedAt = doc.IssuedAt.Format(time.RFC3339)
	}
	if doc.ExpiresAt != nil {
		response.ExpiresAt = doc.ExpiresAt.Format(time.RFC3339)
	}
	if len(decryptedMetadata) > 0 {
		var metadata map[string]any
		if err := json.Unmarshal(decryptedMetadata, &metadata); err == nil {
			response.Metadata = metadata
		}
	}
	if response.Metadata == nil {
		response.Metadata = map[string]any{}
	}
	return response
}

func ToDocumentResponses(docs []model.CitizenDocument, decryptedMetadata [][]byte) []dto.DocumentResponse {
	if len(docs) == 0 {
		return []dto.DocumentResponse{}
	}

	resp := make([]dto.DocumentResponse, 0, len(docs))
	for i, doc := range docs {
		metadata := []byte(nil)
		if i < len(decryptedMetadata) {
			metadata = decryptedMetadata[i]
		}
		resp = append(resp, ToDocumentResponse(doc, metadata))
	}
	return resp
}

func ToUserDocumentResponse(doc model.CitizenDocument, decryptedMetadata []byte) dto.UserDocumentResponse {
	return ToDocumentResponse(doc, decryptedMetadata)
}

func ToUserDocumentResponses(docs []model.CitizenDocument, decryptedMetadata [][]byte) []dto.UserDocumentResponse {
	response := make([]dto.UserDocumentResponse, 0, len(docs))
	for i, doc := range docs {
		metadata := []byte(nil)
		if i < len(decryptedMetadata) {
			metadata = decryptedMetadata[i]
		}
		response = append(response, ToDocumentResponse(doc, metadata))
	}
	return response
}
