package mapper

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"gorm.io/datatypes"
)

func TestToDocumentResponse_IncludesDecryptedMetadataAndTypeCode(t *testing.T) {
	issuedAt := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2028, 1, 2, 0, 0, 0, 0, time.UTC)

	doc := model.CitizenDocument{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		DocumentType: "ID_CARD",
		Status:       model.DocumentStatusActive,
		Metadata:     datatypes.JSON(`{"title":"Dowód osobisty","issuer":"Rzeczpospolita Polska","category":"identity"}`),
		IssuedAt:     &issuedAt,
		ExpiresAt:    &expiresAt,
	}

	resp := ToDocumentResponse(doc)

	if resp.TypeCode != "ID_CARD" {
		t.Fatalf("TypeCode = %q, want %q", resp.TypeCode, "ID_CARD")
	}
	if resp.Metadata == nil {
		t.Fatal("Metadata is nil, want decrypted JSON payload")
	}
	if resp.Metadata["title"] != "Dowód osobisty" {
		t.Fatalf("Metadata[title] = %v, want %q", resp.Metadata["title"], "Dowód osobisty")
	}
	if resp.Status != "ACTIVE" {
		t.Fatalf("Status = %q, want %q", resp.Status, "ACTIVE")
	}
}
