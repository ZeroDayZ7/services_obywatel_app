package shared

import "github.com/google/uuid"

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
