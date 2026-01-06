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
