package model

import (
	"time"

	"github.com/google/uuid"
)

type RegisterCitizenResponse struct {
	UserID               uuid.UUID `json:"user_id"`
	AgreementNumber      string    `json:"agreement_number"`
	AgreementDownloadURL string    `json:"agreement_download_url"`
	PukCode              string    `json:"puk_code"`
	CreatedAt            time.Time `json:"created_at"`
}
