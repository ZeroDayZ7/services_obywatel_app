package shared

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
)

// GenerateCSRFToken zwraca losowy token CSRF
func GenerateUuid() string {
	return uuid.NewString()
}

func GenerateUuidV7() string {
	u, _ := uuid.NewV7()
	return u.String()
}

func MustGenerateUuidV7() uuid.UUID {
	u, err := uuid.NewV7()
	if err != nil {
		// V7 błąd zwraca tylko gdy zegar systemowy cofa się drastycznie
		return uuid.New()
	}
	return u
}

func GenerateRandomChallenge(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Zwracamy Base64, bo łatwo go przesłać w JSON i podpisać we Flutterze
	return base64.StdEncoding.EncodeToString(bytes), nil
}
