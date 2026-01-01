package validator

import (
	"github.com/zerodayz7/platform/services/gateway/internal/middleware"
)

type RegisterRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=3,max=30"`
	Email    string `json:"email" validate:"required,email"`
	Password []byte `json:"password_bytes" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password []byte `json:"password_bytes" validate:"required"`
}

type TwoFARequest struct {
	Code []byte `json:"code_bytes" validate:"required,min=6"`
}

// ===== JWT Refresh =====
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ===== Helpers =====
func ValidateStruct(s any) map[string]string {
	return middleware.ValidateStruct(s)
}
