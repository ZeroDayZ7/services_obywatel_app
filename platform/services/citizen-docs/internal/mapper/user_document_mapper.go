package mapper

import (
	"encoding/base64"
	"time"

	"github.com/zerodayz7/platform/services/citizen-docs/internal/dto"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
)

func ToUserDocumentResponse(doc model.UserDocument) dto.UserDocumentResponse {
	response := dto.UserDocumentResponse{
		ID:               doc.ID.String(),
		TypeCode:         doc.TypeCode,
		Status:           string(doc.Status),
		EncryptedMeta:    base64.StdEncoding.EncodeToString(doc.EncryptedMeta),
		IssuerSignature:  base64.StdEncoding.EncodeToString(doc.IssuerSignature),
		SigningKeyID:     doc.SigningKeyID,
		RevocationSerial: doc.RevocationSerial,
		Version:          doc.Version,
	}

	if len(doc.EncryptedFront) > 0 {
		response.EncryptedFront = base64.StdEncoding.EncodeToString(doc.EncryptedFront)
	}

	if len(doc.EncryptedBack) > 0 {
		response.EncryptedBack = base64.StdEncoding.EncodeToString(doc.EncryptedBack)
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