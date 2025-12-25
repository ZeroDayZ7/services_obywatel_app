package validator

import (
	"github.com/zerodayz7/platform/services/notification-service/internal/middleware"
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
	Code string `json:"code" validate:"required,len=6,numeric"`
}

// ===== JWT Refresh =====
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ===== Helpers =====
func ValidateStruct(s any) map[string]string {
	return middleware.ValidateStruct(s)
}
