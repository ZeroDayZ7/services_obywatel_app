package config

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/crypto"
	"github.com/zerodayz7/platform/pkg/envelope"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const defaultDocumentDataKeyAlias = "docs-id-cards"

func documentDataKeyAlias(documentType string) string {
	switch strings.ToUpper(strings.TrimSpace(documentType)) {
	case "ID_CARD", "ID-CARD":
		return "docs-id-cards"
	case "DRIVERS_LICENSE", "DRIVER_LICENSE", "DRIVERS-LICENSE":
		return "docs-driver-license"
	case "PASSPORT":
		return "docs-passport"
	default:
		return defaultDocumentDataKeyAlias
	}
}

func SeedData(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}

	keyStore := httpserver.NewKeyStore()
	if err := loadSeedSecurityKeys(&AppConfig, keyStore); err != nil {
		return fmt.Errorf("failed to load security keys for seed: %w", err)
	}

	cryptor := envelope.NewEnvelopeCryptor(AppConfig.ToKMSServiceConfig())
	return SeedDataWithSecurity(db, keyStore, cryptor, &AppConfig, false)
}

func loadSeedSecurityKeys(cfg *Config, keyStore *httpserver.KeyStore) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if keyStore == nil {
		return fmt.Errorf("keystore is nil")
	}

	kmsCfg := cfg.ToKMSServiceConfig()
	if err := kms.HealthCheck(context.Background(), kmsCfg); err != nil {
		return fmt.Errorf("kms health check: %w", err)
	}

	for alias, target := range cfg.GetAllSecurityKeys() {
		keyBytes, version, err := kms.FetchSymmetricKeyWithVersion(context.Background(), kmsCfg, target.TargetKey, 1, target.Algorithm)
		if err != nil {
			return fmt.Errorf("fetch key %s (%s): %w", alias, target.TargetKey, err)
		}
		keyStore.SetKey(alias, keyBytes, uint32(version))
	}

	return nil
}

func SeedDataWithSecurity(db *gorm.DB, keyStore *httpserver.KeyStore, cryptor *envelope.EnvelopeCryptor, cfg *Config, forceSeed bool) error {
	log := shared.GetLogger()
	log.Info("[SEED] Rozpoczynam wykonywanie seedera dokumentów obywateli", "force_seed", forceSeed)

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

	if count > 0 && !forceSeed {
		log.Info("[SEED] Tabela dokumentów już zawiera dane; sprawdzam, czy brakuje rekordów do uzupełnienia.", "existing_documents", count)
	} else if count > 0 && forceSeed {
		if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.CitizenDocument{}).Error; err != nil {
			return fmt.Errorf("failed to clear existing documents before forced seed: %w", err)
		}
		log.Info("[SEED] Wymuszony seed aktywny: wyczyszczono istniejące dokumenty przed uzupełnieniem danych testowych.", "removed_documents", count)
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

	seedInputs := []struct {
		UserID         uuid.UUID
		DocumentType   string
		DocumentNumber string
		Status         model.DocumentStatus
		Metadata       datatypes.JSON
		IssuedAt       *time.Time
		ExpiresAt      *time.Time
	}{
		{UserID: citizenUserID1, DocumentType: "ID_CARD", DocumentNumber: "ABC123456", Status: model.DocumentStatusActive, Metadata: datatypes.JSON(`{"issuer":"Prezydent Miasta Katowice","country":"PL","organ_code":"2469"}`), IssuedAt: &issued1YearAgo, ExpiresAt: &expiresIn9Years},
		{UserID: citizenUserID1, DocumentType: "DRIVERS_LICENSE", DocumentNumber: "99999/22/2469", Status: model.DocumentStatusActive, Metadata: datatypes.JSON(`{"issuer":"Starosta Będziński","country":"PL","categories":["B","A2"],"restrictions":"01.06"}`), IssuedAt: &issued5YearsAgo, ExpiresAt: &expiresIn5Years},
		{UserID: citizenUserID1, DocumentType: "PASSPORT", DocumentNumber: "EA8765432", Status: model.DocumentStatusActive, Metadata: datatypes.JSON(`{"issuer":"Wojewoda Śląski","country":"PL","biometric":true}`), IssuedAt: &issuedRecent, ExpiresAt: &expiresIn9Years},
		{UserID: citizenUserID1, DocumentType: "LARGE_FAMILY_CARD", DocumentNumber: "KDR-10203040-01", Status: model.DocumentStatusPending, Metadata: datatypes.JSON(`{"issuer":"Ministerstwo Rodziny i Polityki Społecznej","discount_tier":"STANDARD"}`), IssuedAt: &issuedRecent, ExpiresAt: &expiresIn3Years},
		{UserID: citizenUserID2, DocumentType: "ID_CARD", DocumentNumber: "XYZ987654", Status: model.DocumentStatusExpired, Metadata: datatypes.JSON(`{"issuer":"Burmistrz Sosnowca","country":"PL","renewal_requested":true}`), IssuedAt: &issued10YearsAgo, ExpiresAt: &expired1MonthAgo},
		{UserID: citizenUserID2, DocumentType: "DRIVERS_LICENSE", DocumentNumber: "12345/18/2475", Status: model.DocumentStatusActive, Metadata: datatypes.JSON(`{"issuer":"Prezydent Miasta Sosnowiec","country":"PL","categories":["B","BE","C"]}`), IssuedAt: &issued5YearsAgo, ExpiresAt: &expiresIn5Years},
		{UserID: citizenUserID2, DocumentType: "VEHICLE_REGISTRATION", DocumentNumber: "DR/ABC/0091823", Status: model.DocumentStatusActive, Metadata: datatypes.JSON(`{"issuer":"Wydział Komunikacji Sosnowiec","vin_masked":"WF0XXXGCDX...1234","plate_number":"SO 12345"}`), IssuedAt: &issued1YearAgo},
		{UserID: citizenUserID2, DocumentType: "PASSPORT", DocumentNumber: "EB1122334", Status: model.DocumentStatusRevoked, Metadata: datatypes.JSON(`{"issuer":"Wojewoda Śląski","country":"PL","revocation_reason":"REPORTED_LOST"}`), IssuedAt: &issued5YearsAgo, ExpiresAt: &expired1MonthAgo},
	}

	toInsert := make([]model.CitizenDocument, 0, len(seedInputs))
	for i := range seedInputs {
		documentNumber := seedInputs[i].DocumentNumber
		hash := crypto.ComputeHMAC256Hex([]byte(documentNumber), documentNumberSecret)
		if !forceSeed {
			var existing model.CitizenDocument
			err := db.WithContext(context.Background()).Where("document_number_hash = ?", hash).First(&existing).Error
			if err == nil {
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to check document %s before seed insert: %w", documentNumber, err)
			}
		}

		metadataBytes := []byte(seedInputs[i].Metadata)
		if len(metadataBytes) == 0 {
			metadataBytes = []byte(`{}`)
		}
		keyAlias := documentDataKeyAlias(seedInputs[i].DocumentType)
		encryptedPayload, err := cryptor.SealWithDataKey(context.Background(), keyAlias, metadataBytes)
		if err != nil {
			return fmt.Errorf("failed to encrypt metadata for document %s via KMS alias %s: %w", documentNumber, keyAlias, err)
		}

		toInsert = append(toInsert, model.CitizenDocument{
			UserID:             seedInputs[i].UserID,
			DocumentType:       seedInputs[i].DocumentType,
			Status:             seedInputs[i].Status,
			DocumentNumberHash: hash,
			EncryptedMetadata:  encryptedPayload.EncryptedData,
			EncryptedDEK:       encryptedPayload.EncryptedDEK,
			IssuedAt:           seedInputs[i].IssuedAt,
			ExpiresAt:          seedInputs[i].ExpiresAt,
		})
	}

	if len(toInsert) == 0 {
		log.Info("[SEED] Brak nowych dokumentów do zasiewu. Seed został pominięty, ponieważ wszystkie rekordy są już obecne.", "existing_documents", count)
		return nil
	}

	if err := db.Create(&toInsert).Error; err != nil {
		return fmt.Errorf("failed to seed citizen documents: %w", err)
	}

	log.Info("[SEED] Zakończono zasiewanie dokumentów obywateli", "seeded_documents", len(toInsert), "force_seed", forceSeed)
	return nil
}
