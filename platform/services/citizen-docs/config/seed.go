package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var testUserID = uuid.MustParse("707a8869-6867-4601-9337-e23fcb51b0ad")

func hashPesel(pesel string) string {
	h := hmac.New(sha256.New, []byte("seed-secret-key"))
	h.Write([]byte(pesel))
	return hex.EncodeToString(h.Sum(nil))
}

func SeedData(db *gorm.DB) error {
	log := shared.GetLogger()

	var count int64
	if err := db.Model(&model.CitizenProfile{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check citizen profiles count: %w", err)
	}

	if count > 0 {
		return nil
	}

	log.Info("[SEED] Rozpoczynam zasiewanie bazy citizen-docs...")

	rawCitizenData := model.CitizenData{
		FirstName:   "Jan",
		LastName:    "Kowalski",
		PESEL:       "90010112345",
		DateOfBirth: "1990-01-01",
		Citizenship: "PL",
		Attributes:  datatypes.JSON([]byte(`{"gender":"M","blood_type":"A+"}`)),
	}

	bytesCitizenData, err := json.Marshal(rawCitizenData)
	if err != nil {
		return fmt.Errorf("failed to marshal seed citizen data: %w", err)
	}

	profileID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate profile uuidv7: %w", err)
	}

	now := time.Now()
	issuedAtID := now.AddDate(-2, 0, 0)
	expiresAtID := now.AddDate(8, 0, 0)

	metaIDCard := model.DocumentMeta{
		DocumentNumber: "ABA123456",
		Issuer:         "PREZYDENT MIASTA KATOWICE",
	}
	bytesMetaID, _ := json.Marshal(metaIDCard)

	issuedAtDL := now.AddDate(-1, 0, 0)
	expiresAtDL := now.AddDate(14, 0, 0)

	metaDriverLicense := model.DocumentMeta{
		DocumentNumber: "12345/22/2401",
		Issuer:         "STAROSTA BĘDZIŃSKI",
		AdditionalInfo: datatypes.JSON([]byte(`{"categories":["AM","B"]}`)),
	}
	bytesMetaDL, _ := json.Marshal(metaDriverLicense)

	docID1, _ := uuid.NewV7()
	docID2, _ := uuid.NewV7()

	profile := model.CitizenProfile{
		ID:            profileID,
		UserID:        testUserID,
		EncryptedData: bytesCitizenData,
		PeselHash:     hashPesel("90010112345"),
		Documents: []model.UserDocument{
			{
				ID:            docID1,
				Type:          model.DocumentTypeIDCard,
				Status:        model.DocumentStatusActive,
				EncryptedMeta: bytesMetaID,
				IssuedAt:      &issuedAtID,
				ExpiresAt:     &expiresAtID,
			},
			{
				ID:            docID2,
				Type:          model.DocumentTypeDriverLicense,
				Status:        model.DocumentStatusActive,
				EncryptedMeta: bytesMetaDL,
				IssuedAt:      &issuedAtDL,
				ExpiresAt:     &expiresAtDL,
			},
		},
	}

	if err := db.Create(&profile).Error; err != nil {
		return fmt.Errorf("failed to seed citizen profile with documents: %w", err)
	}

	log.Info(fmt.Sprintf("[SEED] Utworzono profil obywatela: %s | ID Profilu: %s | Dokumenty: %d", profile.UserID, profile.ID, len(profile.Documents)))
	return nil
}
