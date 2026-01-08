package errors

// ErrorType definiuje typ błędu
type ErrorType string

const (
	Unauthorized ErrorType = "UNAUTHORIZED"
	Validation   ErrorType = "VALIDATION"
	NotFound     ErrorType = "NOT_FOUND"
	Internal     ErrorType = "INTERNAL"
	BadRequest   ErrorType = "BAD_REQUEST"
)

// Domyślne komunikaty dla typów błędów
var ErrorMessages = map[ErrorType]string{
	Unauthorized: "Brak autoryzacji.",
	Validation:   "Nieprawidłowe dane.",
	NotFound:     "Zasób nie został znaleziony.",
	Internal:     "Wewnętrzny błąd serwera.",
	BadRequest:   "Błędne żądanie.",
}

// AppError to baza dla wszystkich błędów serwisów
type AppError struct {
	Code    string
	Type    ErrorType
	Message string
	Err     error
	Meta    map[string]any
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return ErrorMessages[e.Type]
}

// --- Uniwersalne błędy ---
var (
	ErrInternal         = &AppError{Code: "SERVER_ERROR", Type: Internal, Message: "Internal server error"}
	ErrInvalidJSON      = &AppError{Code: "INVALID_JSON", Type: BadRequest, Message: "Invalid JSON in request body"}
	ErrValidationFailed = &AppError{Code: "VALIDATION_FAILED", Type: Validation, Message: "Request validation failed"}
	ErrTooManyRequests  = &AppError{Code: "TOO_MANY_REQUESTS", Type: BadRequest, Message: "Too many requests"}
	ErrUnauthorized     = &AppError{Code: "UNAUTHORIZED", Type: Unauthorized, Message: "Unauthorized access"}
	ErrInvalidToken     = &AppError{Code: "INVALID_TOKEN", Type: Unauthorized, Message: "Invalid token"}
)

// --- Błędy specyficzne dla auth ---
var (
	ErrInvalid2FACode      = &AppError{Code: "INVALID_2FA", Type: Validation, Message: "Invalid 2FA code"}
	ErrInvalidRequest      = &AppError{Code: "INVALID_REQUEST", Type: Validation, Message: "Invalid request data"}
	ErrEmailExists         = &AppError{Code: "EMAIL_EXISTS", Type: Validation, Message: "Email already registered"}
	ErrUsernameExists      = &AppError{Code: "USERNAME_EXISTS", Type: Validation, Message: "Username already exist"}
	ErrPasswordTooShort    = &AppError{Code: "PASSWORD_TOO_SHORT", Type: Validation, Message: "Password must be at least 8 characters"}
	ErrCSRFInvalid         = &AppError{Code: "CSRF_INVALID", Type: Unauthorized, Message: "CSRF token invalid or missing"}
	ErrInvalidCredentials  = &AppError{Code: "INVALID_CREDENTIALS", Type: Unauthorized, Message: "Incorrect login data"}
	ErrUserNotFound        = &AppError{Code: "USER_NOT_FOUND", Type: Unauthorized, Message: "User not found"}
	ErrEmailIsSendIfExists = &AppError{Code: "EMAIL_IS_SEND_IF_EXISTS", Type: Validation, Message: "If the account exists, a reset code has been sent to the provided email."}
	ErrAccountLocked       = &AppError{Code: "ACCOUNT_LOCKED", Type: Unauthorized, Message: "Account locked due to too many failed login attempts"}
	Err2FALocked           = &AppError{Code: "2FA_LOCKED", Type: Validation, Message: "Too many incorrect 2FA attempts. Try again in 15 minutes."}
	ErrSessionExpired      = &AppError{Code: "SESSION_EXPIRED", Type: Unauthorized, Message: "Pairing session expired. Please log in again."}
	ErrVerificationFailed  = &AppError{Code: "VERIFICATION_FAILED", Type: Unauthorized, Message: "Device verification failed."}
	ErrInvalidPairingData  = &AppError{Code: "INVALID_PAIRING_DATA", Type: Validation, Message: "Invalid public key or signature format"}
	ErrAccountSuspended    = &AppError{Code: "ACCOUNT_SUSPENDED", Type: Unauthorized, Message: "Konto użytkownika jest tymczasowo zawieszone."}
	ErrAccountBanned       = &AppError{Code: "ACCOUNT_BANNED", Type: Unauthorized, Message: "Konto użytkownika zostało zablokowane."}
	ErrAccountPending      = &AppError{Code: "ACCOUNT_PENDING", Type: Unauthorized, Message: "Konto użytkownika oczekuje na weryfikację."}
)

var (
	ErrResetSessionNotFound = &AppError{Code: "RESET_SESSION_NOT_FOUND", Type: BadRequest, Message: "Sesja resetowania hasła wygasła lub jest nieprawidłowa."}
	ErrInvalidResetCode     = &AppError{Code: "INVALID_RESET_CODE", Type: Validation, Message: "Nieprawidłowy kod resetujący."}
	ErrUntrustedDevice      = &AppError{Code: "UNTRUSTED_DEVICE", Type: Unauthorized, Message: "To urządzenie nie jest zaufane dla tego konta."}
	ErrInvalidSignature     = &AppError{Code: "INVALID_SIGNATURE", Type: Unauthorized, Message: "Nieprawidłowy podpis bezpieczeństwa."}
)
