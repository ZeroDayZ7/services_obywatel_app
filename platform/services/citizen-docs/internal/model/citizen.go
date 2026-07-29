package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// BaseModel – Standardowe pola audytowe z rozszerzonym wsparciem dla wersji
type BaseModel struct {
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type DocumentStatus string

const (
	DocumentStatusActive    DocumentStatus = "active"
	DocumentStatusInactive  DocumentStatus = "inactive"
	DocumentStatusExpired   DocumentStatus = "expired"
	DocumentStatusRevoked   DocumentStatus = "revoked"
	DocumentStatusPending   DocumentStatus = "pending"
	DocumentStatusSuspended DocumentStatus = "suspended"
)

// CitizenProfile – Główny profil obywatela w mikroserwisie dokumentów
type CitizenProfile struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID        uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"` // Identyfikator z Auth Service
	EncryptedData []byte    `gorm:"type:bytea;not null"`            // Zaszyfrowana struktura CitizenData
	PeselHash     string    `gorm:"size:64;uniqueIndex;not null"`   // Salted HMAC-SHA256 z PESEL-u

	// Wersjonowanie do szybkiej synchronizacji różnicowej z aplikacją mobilną
	Version uint64 `gorm:"not null;default:1;index"`

	// Relacja wewnątrz bazy mikroserwisu
	Documents []UserDocument `gorm:"foreignKey:ProfileID" json:"documents,omitempty"`

	BaseModel
}

// UserDocument – Generyczna encja reprezentująca dowolny dokument
type UserDocument struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	ProfileID uuid.UUID `gorm:"type:uuid;index;not null"`

	// Typ dokumentu jako elastyczny ciąg tekstowy (np. "id_card", "passport", "city_transit_pass", "ticket")
	TypeCode string         `gorm:"type:varchar(64);not null;index"`
	Status   DocumentStatus `gorm:"type:varchar(20);not null;default:'active';index"`

	// Zaszyfrowane ładunki (Payloads)
	EncryptedMeta  []byte `gorm:"type:bytea;not null"` // Zaszyfrowana struktura DocumentMeta
	EncryptedFront []byte `gorm:"type:bytea" json:"-"` // Opcjonalny wizerunek / szablon graficzny frontu
	EncryptedBack  []byte `gorm:"type:bytea" json:"-"` // Opcjonalny wizerunek / szablon graficzny tyłu

	// Podpis Cyfrowy Wystawcy (Cryptographic Proof dla trybu Offline QR)
	// Wygenerowany przez Crypto Engine w Rucie przy użyciu klucza prywatnego systemu
	IssuerSignature []byte `gorm:"type:bytea;not null"`
	SigningKeyID    string `gorm:"type:varchar(64);not null;index"` // Identyfikator klucza publicznego (KMS/PKI)

	// Identyfikator unieważnienia (dla szybkich sprawdzeń na liście CRL/Revocation)
	RevocationSerial string `gorm:"type:varchar(128);uniqueIndex;not null"`

	// Daty ważności i wydania (jawne w DB dla szybkich kwerend indeksowanych i background workerów)
	IssuedAt  *time.Time `gorm:"index"`
	ExpiresAt *time.Time `gorm:"index"`

	// Wersjonowanie poszczególnego dokumentu dla delty synchronizacji z aplikacją mobilną
	Version uint64 `gorm:"not null;default:1;index"`

	// Relacja z profilem obywatela (Cascade Delete przy usunięciu konta)
	Profile *CitizenProfile `gorm:"foreignKey:ProfileID;constraint:OnDelete:CASCADE"`

	BaseModel
}

// --- Struktury pamięciowe (In-Memory / DTOs po odszyfrowaniu) ---

// CitizenData – Odszyfrowane dane osobowe profilu
type CitizenData struct {
	FirstName   string         `json:"first_name"`
	LastName    string         `json:"last_name"`
	PESEL       string         `json:"pesel"`
	DateOfBirth string         `json:"date_of_birth"`
	Citizenship string         `json:"citizenship"`
	Attributes  datatypes.JSON `json:"attributes,omitempty"` // Dynamiczne atrybuty (np. organ wydający profil, status niepełnosprawności)
}

// DocumentMeta – Odszyfrowane metadane dokumentu (obsługuje dowolne typy od dowodu po bilet)
type DocumentMeta struct {
	DocumentNumber string `json:"document_number"`
	Title          string `json:"title,omitempty"`    // np. "Warszawska Karta Miejska", "Bilet Okresowy PKP"
	Issuer         string `json:"issuer,omitempty"`   // np. "Prezydent M.St. Warszawy", "Minister Cyfryzacji"
	Category       string `json:"category,omitempty"` // np. "identity", "transport", "qualification", "health"

	// Definicja dostępnych widoków / zakresów weryfikacji w kodzie QR (Offline Selective Disclosure)
	// Przykłady: ["age_verification", "full_identity", "transit_gate_check"]
	AllowedScopes []string `json:"allowed_scopes,omitempty"`

	// Generyczne pola dynamiczne dla specyficznych dokumentów
	// np. dla prawa jazdy: {"categories": ["B", "A"]}, dla biletu: {"zone": "A+B", "line": "M1"}
	CustomAttributes datatypes.JSON `json:"custom_attributes,omitempty"`
}

// DocumentVerificationPayload – Struktura generowana i pakowana do kodu QR w aplikacji mobilnej (Offline)
type DocumentVerificationPayload struct {
	DocumentID       uuid.UUID      `json:"doc_id"`
	ProfileID        uuid.UUID      `json:"profile_id"`
	RevocationSerial string         `json:"revocation_serial"`
	Scope            string         `json:"scope"`      // Zakres wybranej weryfikacji (np. "age_verification")
	Claims           datatypes.JSON `json:"claims"`     // Wycięte, selektywne dane (np. tylko rok urodzenia)
	IssuedNonce      int64          `json:"nonce"`      // Timestamp wygenerowania QR (np. ważny 3 minuty)
	IssuerSignature  []byte         `json:"issuer_sig"` // Oryginalny podpis backendu z pola UserDocument.IssuerSignature
	DeviceSignature  []byte         `json:"device_sig"` // Podpis z klucza prywatnego urządzenia (Secure Enclave / Android Keystore)
}
