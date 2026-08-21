package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"gorm.io/gorm"
)

// Struktura pod plik JSON dla Angulara
type AngularDevCard struct {
	CardSerialNumber string `json:"cardSerialNumber"`
	PublicKey        string `json:"publicKey"`
	PrivateKey       string `json:"privateKey"`
	UserID           string `json:"userId"`
}

func SeedInitialEmployeeCredential(db *gorm.DB) error {
	log := shared.GetLogger()

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Error("❌ Błąd podczas generowania kluczy Ed25519", "error", err)
		return err
	}

	pubKeyHex := hex.EncodeToString(pubKey)
	privKeyHex := hex.EncodeToString(privKey)

	adminUserID := uuid.MustParse("92b98b5a-d0c3-410f-828d-2b30a585dea6")
	systemIssuerID := uuid.MustParse("707a8869-6867-4601-9337-e23fcb51b0ad")

	credential := model.EmployeeCredential{
		ID:               uuid.New(),
		UserID:           adminUserID,
		CardSerialNumber: "CARD-OFFICER-DEV-001",
		PublicKey:        pubKeyHex,
		KeyAlgorithm:     "ED25519",
		Status:           model.EmployeeCredentialActive,
		IssuedBy:         systemIssuerID,
		ExpiresAt:        func() *time.Time { t := time.Now().AddDate(1, 0, 0); return &t }(),
	}

	err = db.Where("card_serial_number = ?", credential.CardSerialNumber).FirstOrCreate(&credential).Error
	if err != nil {
		log.Error("❌ Błąd zapisywania poświadczenia w bazie", "error", err)
		return err
	}

	// Zapis do pliku dla Angulara
	devCard := AngularDevCard{
		CardSerialNumber: credential.CardSerialNumber,
		PublicKey:        pubKeyHex,
		PrivateKey:       privKeyHex,
		UserID:           adminUserID.String(),
	}

	fileData, err := json.MarshalIndent(devCard, "", "  ")
	if err == nil {
		_ = os.WriteFile("dev-card.json", fileData, 0644)
		log.Info("✅ Zaseedowano poświadczenie urzędnika (zapisano w pliku dev-card.json)", "card_serial", credential.CardSerialNumber)
	} else {
		log.Warn("⚠️ Nie udało się zapisać pliku dev-card.json", "error", err)
	}

	return nil
}
