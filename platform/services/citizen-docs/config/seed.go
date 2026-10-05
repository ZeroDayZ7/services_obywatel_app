package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/crypto"
	"github.com/zerodayz7/platform/pkg/envelope"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func SeedData(db *gorm.DB) error {
	return SeedDataWithSecurity(db, nil, nil, nil)
}

func SeedDataWithSecurity(db *gorm.DB, keyStore *httpserver.KeyStore, cryptor *envelope.EnvelopeCryptor, cfg *Config) error {
	log := shared.GetLogger()
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	if keyStore == nil || cryptor == nil {
		return fmt.Errorf("document seeding requires a valid KeyStore and EnvelopeCryptor")
	}

	var count int64
	if err := db.Model(&model.CitizenDocument{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check document count: %w", err)
	}
	if count > 0 {
		return nil
	}

	documentNumberSecret, _, ok := keyStore.GetKey("document_number")
	if !ok {
		return fmt.Errorf("missing 'document_number' HMAC key in KeyStore")
	}

	citizenUserID1 := uuid.MustParse("a2f6b8c9-1122-4a55-8822-b98765432101")
	citizenUserID2 := uuid.MustParse("c3d4e5f6-3344-5b66-9933-a12345678902")

	now := time.Now()
	issued5YearsAgo := now.AddDate(-5, 0, 0)
	expiresIn5Years := now.AddDate(5, 0, 0)
	issued1YearAgo := now.AddDate(-1, 0, 0)
	expiresIn9Years := now.AddDate(9, 0, 0)
	issued10YearsAgo := now.AddDate(-10, 0, 0)
	expired1MonthAgo := now.AddDate(0, -1, 0)
	issuedRecent := now.AddDate(0, -2, 0)
	expiresIn3Years := now.AddDate(3, 0, 0)

	seedDocs := []model.CitizenDocument{
		{UserID: citizenUserID1, DocumentType: "ID_CARD", DocumentNumber: "ABC123456", Status: model.DocumentStatusActive, Metadata: datatypes.JSON(`{"issuer":"Prezydent Miasta Katowice","country":"PL","organ_code":"2469"}`), IssuedAt: &issued1YearAgo, ExpiresAt: &expiresIn9Years},
		{UserID: citizenUserID1, DocumentType: "DRIVERS_LICENSE", DocumentNumber: "99999/22/2469", Status: model.DocumentStatusActive, Metadata: datatypes.JSON(`{"issuer":"Starosta Będziński","country":"PL","categories":["B","A2"],"restrictions":"01.06"}`), IssuedAt: &issued5YearsAgo, ExpiresAt: &expiresIn5Years},
		{UserID: citizenUserID1, DocumentType: "PASSPORT", DocumentNumber: "EA8765432", Status: model.DocumentStatusActive, Metadata: datatypes.JSON(`{"issuer":"Wojewoda Śląski","country":"PL","biometric":true}`), IssuedAt: &issuedRecent, ExpiresAt: &expiresIn9Years},
		{UserID: citizenUserID1, DocumentType: "LARGE_FAMILY_CARD", DocumentNumber: "KDR-10203040-01", Status: model.DocumentStatusPending, Metadata: datatypes.JSON(`{"issuer":"Ministerstwo Rodziny i Polityki Społecznej","discount_tier":"STANDARD"}`), IssuedAt: &issuedRecent, ExpiresAt: &expiresIn3Years},
		{UserID: citizenUserID2, DocumentType: "ID_CARD", DocumentNumber: "XYZ987654", Status: model.DocumentStatusExpired, Metadata: datatypes.JSON(`{"issuer":"Burmistrz Sosnowca","country":"PL","renewal_requested":true}`), IssuedAt: &issued10YearsAgo, ExpiresAt: &expired1MonthAgo},
		{UserID: citizenUserID2, DocumentType: "DRIVERS_LICENSE", DocumentNumber: "12345/18/2475", Status: model.DocumentStatusActive, Metadata: datatypes.JSON(`{"issuer":"Prezydent Miasta Sosnowiec","country":"PL","categories":["B","BE","C"]}`), IssuedAt: &issued5YearsAgo, ExpiresAt: &expiresIn5Years},
		{UserID: citizenUserID2, DocumentType: "VEHICLE_REGISTRATION", DocumentNumber: "DR/ABC/0091823", Status: model.DocumentStatusActive, Metadata: datatypes.JSON(`{"issuer":"Wydział Komunikacji Sosnowiec","vin_masked":"WF0XXXGCDX...1234","plate_number":"SO 12345"}`), IssuedAt: &issued1YearAgo},
		{UserID: citizenUserID2, DocumentType: "PASSPORT", DocumentNumber: "EB1122334", Status: model.DocumentStatusRevoked, Metadata: datatypes.JSON(`{"issuer":"Wojewoda Śląski","country":"PL","revocation_reason":"REPORTED_LOST"}`), IssuedAt: &issued5YearsAgo, ExpiresAt: &expired1MonthAgo},
	}

	for i := range seedDocs {
		metadataBytes := []byte(seedDocs[i].Metadata)
		if len(metadataBytes) == 0 {
			metadataBytes = []byte(`{}`)
		}
		encryptedPayload, err := cryptor.SealWithDataKey(context.Background(), "documents-metadata-key", metadataBytes)
		if err != nil {
			return fmt.Errorf("failed to encrypt metadata for document %s: %w", seedDocs[i].DocumentNumber, err)
		}
		seedDocs[i].DocumentNumberHash = crypto.ComputeHMAC256Hex([]byte(seedDocs[i].DocumentNumber), documentNumberSecret)
		seedDocs[i].EncryptedMetadata = encryptedPayload.EncryptedData
		seedDocs[i].EncryptedDEK = encryptedPayload.EncryptedDEK
		seedDocs[i].Metadata = datatypes.JSON(metadataBytes)
	}

	if err := db.Create(&seedDocs).Error; err != nil {
		return fmt.Errorf("failed to seed citizen documents: %w", err)
	}

	log.Info("[SEED] Pomyślnie zasiano zróżnicowany zestaw dokumentów obywateli z wykorzystaniem Envelope Encryption i HMAC.")
	return nil
}

func ShouldSeedData(args []string) bool {
	for _, arg := range args {
		if strings.TrimSpace(arg) == "--seed" {
			return true
		}
	}
	seedFlag := strings.TrimSpace(os.Getenv("SEED_DATA"))
	return strings.EqualFold(seedFlag, "true") || strings.EqualFold(seedFlag, "1") || strings.EqualFold(seedFlag, "yes")
}
