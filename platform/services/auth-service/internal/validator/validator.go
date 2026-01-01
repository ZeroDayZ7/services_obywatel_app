package validator

import (
	"github.com/zerodayz7/platform/services/auth-service/internal/middleware"
)

// ===== Reset Password =====
type ResetPasswordRequest struct {
	Value  string `json:"value" validate:"required,email"` // zamiast Email
	Method string `json:"method"`                          // np. "email" lub "sms"
}

type ResetCodeVerifyRequest struct {
	Token string `json:"token" validate:"required"` // token z SendResetCode
	Code  string `json:"code" validate:"required"`  // kod, który użytkownik otrzymał
}

type ResetPasswordFinalRequest struct {
	Token       string `json:"token" validate:"required"`        // token z VerifyResetCode
	NewPassword string `json:"new_password" validate:"required"` // nowe hasło
}

// ===== Registration & Login =====
type RegisterRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=3,max=30"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,passwd"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password []byte `json:"password" validate:"required"`
}

type TwoFARequest struct {
	Code  []byte `json:"code" validate:"required,min=6,max=6"`
	Token string `json:"token" validate:"required"`
}

// ===== JWT Refresh =====
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ===== Helpers =====
func ValidateStruct(s any) map[string]string {
	return middleware.ValidateStruct(s)
}
