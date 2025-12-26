package validator

import (
	"github.com/zerodayz7/platform/services/auth-service/internal/middleware"
)

type RegisterRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=3,max=30"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,passwd"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,passwd"`
}

type TwoFARequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,len=6,numeric"`
	Token string `json:"token" validate:"required,uuid4"`
}

// ===== JWT Refresh =====
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ===== Helpers =====
func ValidateStruct(s any) map[string]string {
	return middleware.ValidateStruct(s)
}
