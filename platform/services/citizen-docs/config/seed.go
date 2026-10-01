package config

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func SeedData(db *gorm.DB) error {
	log := shared.GetLogger()

	var count int64
	if err := db.Model(&model.CitizenDocument{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check document count: %w", err)
	}
	if count > 0 {
		return nil
	}

	userID1 := uuid.MustParse("707a8869-6867-4601-9337-e23fcb51b0ad")
	userID2 := uuid.MustParse("a2f6b8c9-1122-4a55-8822-b98765432101")

	issuedAt1 := time.Now().AddDate(-2, 0, 0)
	expiresAt1 := time.Now().AddDate(8, 0, 0)
	issuedAt2 := time.Now().AddDate(-1, 0, 0)
	expiresAt2 := time.Now().AddDate(4, 0, 0)

	docs := []model.CitizenDocument{
		{
			UserID:         userID1,
			DocumentType:   "ID_CARD",
			DocumentNumber: "PL-ID-10001",
			Status:         model.DocumentStatusActive,
			Metadata:       datatypes.JSON(`{"issuer":"miejskie","country":"PL"}`),
			IssuedAt:       &issuedAt1,
			ExpiresAt:      &expiresAt1,
		},
		{
			UserID:         userID1,
			DocumentType:   "DRIVERS_LICENSE",
			DocumentNumber: "PL-DL-22001",
			Status:         model.DocumentStatusPending,
			Metadata:       datatypes.JSON(`{"issuer":"starostwo","country":"PL","category":"B"}`),
			IssuedAt:       &issuedAt2,
			ExpiresAt:      &expiresAt2,
		},
		{
			UserID:         userID2,
			DocumentType:   "ID_CARD",
			DocumentNumber: "PL-ID-20002",
			Status:         model.DocumentStatusExpired,
			Metadata:       datatypes.JSON(`{"issuer":"urzad","country":"PL"}`),
			IssuedAt:       &issuedAt2,
			ExpiresAt:      &expiresAt2,
		},
	}

	if err := db.Create(&docs).Error; err != nil {
		return fmt.Errorf("failed to seed citizen documents: %w", err)
	}

	log.Info("[SEED] Seeded citizen documents without any PII data.")
	return nil
}
