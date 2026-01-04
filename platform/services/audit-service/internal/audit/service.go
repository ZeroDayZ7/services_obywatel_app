package audit

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"

	"github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/audit-service/db/dbgen"
)

type AuditService struct {
	queries dbgen.Querier
	logger  *shared.Logger
}

func NewAuditService(q dbgen.Querier, l *shared.Logger) *AuditService {
	return &AuditService{
		queries: q,
		logger:  l,
	}
}

// SaveLog teraz przyjmuje strukturę AuditMessage
func (s *AuditService) SaveLog(ctx context.Context, msg AuditMessage) error {
	publicKeyPEM := viper.GetString("ADMIN_PUBLIC_KEY")
	if publicKeyPEM == "" {
		return errors.New("admin public key not found in config")
	}

	// 1. Konwersja Metadata (map) na JSON (bytes), który zostanie zaszyfrowany
	metadataBytes, err := json.Marshal(msg.Metadata)
	if err != nil {
		s.logger.ErrorObj("Failed to marshal metadata", err)
		return err
	}

	// 2. Szyfrowanie hybrydowe (tylko wrażliwe dane z metadata)
	encData, encKey, err := s.encryptHybrid(metadataBytes, publicKeyPEM)
	if err != nil {
		s.logger.ErrorObj("Encryption failed", err)
		return err
	}

	// 3. Zapis przez sqlc - przekazujemy nowe pola: ServiceName i IpAddress
	err = s.queries.CreateEncryptedLog(ctx, dbgen.CreateEncryptedLogParams{
		UserID:        msg.UserID,
		ServiceName:   msg.Service,
		Action:        msg.Action,
		IpAddress:     msg.IPAddress,
		EncryptedData: encData,
		EncryptedKey:  encKey,
		Status:        "SUCCESS",
	})

	if err != nil {
		s.logger.ErrorObj("Failed to save encrypted log to DB", err)
		return err
	}

	return nil
}

// --- Metody dla Handlera (Odczyt zaszyfrowanych danych) ---

func (s *AuditService) GetAllLogs(ctx context.Context, limit, offset int32) ([]dbgen.EncryptedAuditLog, error) {
	return s.queries.GetAllLogs(ctx, dbgen.GetAllLogsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *AuditService) GetLogsByAction(ctx context.Context, action string) ([]dbgen.EncryptedAuditLog, error) {
	// Pamiętaj, aby dodać zapytanie GetLogsByAction do query.sql i odpalić sqlc generate
	return s.queries.GetLogsByAction(ctx, action)
}

func (s *AuditService) GetLogsByUserID(ctx context.Context, userID int64) ([]dbgen.EncryptedAuditLog, error) {
	return s.queries.GetLogsByUserId(ctx, userID)
}

func (s *AuditService) GetLogByID(ctx context.Context, id int64) (dbgen.EncryptedAuditLog, error) {
	// To zapytanie musisz dodać do query.sql i wygenerować
	return s.queries.GetLogByID(ctx, id)
}

// --- Logika Kryptograficzna (Private) ---

func (s *AuditService) encryptHybrid(plainText []byte, pubKeyPEM string) ([]byte, []byte, error) {
	// A. Parsowanie klucza RSA Admina
	block, _ := pem.Decode([]byte(pubKeyPEM))
	if block == nil {
		return nil, nil, errors.New("failed to parse PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, nil, errors.New("not an RSA public key")
	}

	// B. Generowanie losowego klucza AES-256 (32 bajty)
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, nil, err
	}

	// C. Szyfrowanie danych przez AES-GCM (Szybkie)
	blockAes, _ := aes.NewCipher(aesKey)
	gcm, _ := cipher.NewGCM(blockAes)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	// Wynik: nonce + zaszyfrowane dane
	encryptedData := gcm.Seal(nonce, nonce, plainText, nil)

	// D. Szyfrowanie klucza AES przez RSA Admina (Bezpieczne)
	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, aesKey, nil)
	if err != nil {
		return nil, nil, err
	}

	return encryptedData, encryptedKey, nil
}
