package shared

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

// GenerateUuid zwraca standardowe UUID v4
func GenerateUuid() string {
	return uuid.NewString()
}

// GenerateUuidV7 zwraca UUID v7 jako string
func GenerateUuidV7() string {
	u, _ := uuid.NewV7()
	return u.String()
}

// MustGenerateUuidV7 zwraca obiekt uuid.UUID v7
func MustGenerateUuidV7() uuid.UUID {
	u, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return u
}

// GenerateSecureOTP generuje bezpieczny kryptograficznie kod 6-cyfrowy (np. 012345)
func GenerateSecureOTP() (string, error) {
	// Zakres: 000000 - 999999
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	// %06d zapewnia, że kody zaczynające się od zera (np. 007234) nie zostaną ucięte
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// GenerateRandomChallenge generuje losowy challenge w formacie Base64
func GenerateRandomChallenge(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// GenerateSessionID to alias dla UUID v7 – idealny dla kluczy w Redis
func GenerateSessionID() string {
	return GenerateUuidV7()
}
