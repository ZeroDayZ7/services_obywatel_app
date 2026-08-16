package model

import (
	"time"

	"github.com/google/uuid"
)

type RegisterCitizenRequest struct {
	PESEL     string `json:"pesel"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

type RegisterCitizenResponse struct {
	CitizenID      uuid.UUID `json:"citizen_id"`
	PUKCode        string    `json:"puk_code"`
	ActivationCode string    `json:"activation_code"`
	RegisteredAt   time.Time `json:"registered_at"`
}
