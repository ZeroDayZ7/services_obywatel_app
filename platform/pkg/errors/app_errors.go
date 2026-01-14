package errors

import "maps"

// ErrorType definiuje typ błędu
type ErrorType string

const (
	Unauthorized ErrorType = "UNAUTHORIZED"
	Validation   ErrorType = "VALIDATION"
	NotFound     ErrorType = "NOT_FOUND"
	Internal     ErrorType = "INTERNAL"
	BadRequest   ErrorType = "BAD_REQUEST"
	Timeout      ErrorType = "TIMEOUT"
	Conflict     ErrorType = "CONFLICT"
)

// Domyślne komunikaty dla typów błędów
var ErrorMessages = map[ErrorType]string{
	Unauthorized: "Brak autoryzacji.",
	Validation:   "Nieprawidłowe dane.",
	NotFound:     "Zasób nie został znaleziony.",
	Internal:     "Wewnętrzny błąd serwera.",
	BadRequest:   "Błędne żądanie.",
	Timeout:      "Przekroczono czas oczekiwania.",
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
	if msg, ok := ErrorMessages[e.Type]; ok {
		return msg
	}
	return "An unknown error occurred"
}

func (e AppError) WithMeta(key string, value any) *AppError {
	newMeta := maps.Clone(e.Meta)

	if newMeta == nil {
		newMeta = make(map[string]any)
	}

	newMeta[key] = value

	return &AppError{
		Code:    e.Code,
		Type:    e.Type,
		Message: e.Message,
		Meta:    newMeta,
	}
}

func newErr(code string, errType ErrorType, msg string) *AppError {
	return &AppError{Code: code, Type: errType, Message: msg}
}

// --- Uniwersalne błędy ---
var (
	ErrNotFound                  = newErr("NOT_FOUND", NotFound, "Zasób nie został znaleziony.")
	ErrInternal                  = newErr("SERVER_ERROR", Internal, "Internal server error")
	ErrInvalidJSON               = newErr("INVALID_JSON", BadRequest, "Invalid JSON in request body")
	ErrInvalidParams             = newErr("INVALID_PARAMS", BadRequest, "Invalid or missing path parameters")
	ErrInvalidQuery              = newErr("INVALID_QUERY", BadRequest, "Invalid or missing query string parameters")
	ErrValidationFailed          = newErr("VALIDATION_FAILED", Validation, "Request validation failed")
	ErrTooManyRequests           = newErr("TOO_MANY_REQUESTS", BadRequest, "Too many requests")
	ErrUnauthorized              = newErr("UNAUTHORIZED", Unauthorized, "Unauthorized access")
	ErrInvalidToken              = newErr("INVALID_TOKEN", Unauthorized, "Invalid token")
	ErrGatewayTimeout            = newErr("GATEWAY_TIMEOUT", Timeout, "Usługa nie odpowiedziała w wymaganym czasie.")
	ErrUpstreamUnavailable       = newErr("UPSTREAM_UNAVAILABLE", Internal, "Usługa zewnętrzna jest niedostępna.")
	ErrInvalidDeviceFingerprint  = newErr("INVALID_FINGERPRINT", BadRequest, "Identification failed: Missing device fingerprint")
	ErrUpstreamTimeout           = newErr("UPSTREAM_TIMEOUT", Timeout, "Upstream service timeout")
	ErrUpstreamUnreachable       = newErr("UPSTREAM_UNREACHABLE", Internal, "Upstream service unreachable")
	ErrInternalContextEncoding   = newErr("INTERNAL_CONTEXT_ENCODING", Unauthorized, "Błąd kodowania kontekstu wewnętrznego.")
	ErrInternalInvalidSignature  = newErr("INTERNAL_INVALID_SIGNATURE", Unauthorized, "Nieprawidłowa sygnatura wewnętrzna.")
	ErrInternalContextCorruption = newErr("INTERNAL_CONTEXT_CORRUPTION", Unauthorized, "Uszkodzony kontekst wewnętrzny.")
	ErrInvalidRequestBody        = newErr("INVALID_REQUEST_BODY", BadRequest, "Nieprawidłowy format treści żądania.")
	ErrInvalidSession            = newErr("INVALID_SESSION", Unauthorized, "Nieprawidłowa lub niekompletna sesja urządzenia.")
	ErrInvalidChallenge          = newErr("INVALID_CHALLENGE", Unauthorized, "Challenge wygasł lub jest nieprawidłowy.")
)

// --- Błędy specyficzne dla auth ---
var (
	ErrInvalid2FACode      = newErr("INVALID_2FA", Validation, "Invalid 2FA code")
	ErrInvalidRequest      = newErr("INVALID_REQUEST", Validation, "Invalid request data")
	ErrEmailExists         = newErr("EMAIL_EXISTS", Validation, "Email already registered")
	ErrUsernameExists      = newErr("USERNAME_EXISTS", Validation, "Username already exist")
	ErrPasswordTooShort    = newErr("PASSWORD_TOO_SHORT", Validation, "Password must be at least 8 characters")
	ErrCSRFInvalid         = newErr("CSRF_INVALID", Unauthorized, "CSRF token invalid or missing")
	ErrInvalidCredentials  = newErr("INVALID_CREDENTIALS", Unauthorized, "Incorrect login data")
	ErrUserNotFound        = newErr("USER_NOT_FOUND", Unauthorized, "User not found")
	ErrEmailIsSendIfExists = newErr("EMAIL_IS_SEND_IF_EXISTS", Validation, "If the account exists, a reset code has been sent.")
	ErrAccountLocked       = newErr("ACCOUNT_LOCKED", Unauthorized, "Account locked due to too many failed login attempts")
	Err2FALocked           = newErr("2FA_LOCKED", Validation, "Too many incorrect 2FA attempts. Try again in 15 minutes.")
	ErrSessionExpired      = newErr("SESSION_EXPIRED", Unauthorized, "Pairing session expired. Please log in again.")
	ErrVerificationFailed  = newErr("VERIFICATION_FAILED", Unauthorized, "Device verification failed.")
	ErrInvalidPairingData  = newErr("INVALID_PAIRING_DATA", Validation, "Invalid public key or signature format")
	ErrAccountSuspended    = newErr("ACCOUNT_SUSPENDED", Unauthorized, "Konto użytkownika jest tymczasowo zawieszone.")
	ErrAccountBanned       = newErr("ACCOUNT_BANNED", Unauthorized, "Konto użytkownika zostało zablokowane.")
	ErrAccountPending      = newErr("ACCOUNT_PENDING", Unauthorized, "Konto użytkownika oczekuje na weryfikację.")
)

// --- Dodatkowe błędy ---
var (
	ErrResetSessionNotFound = newErr("RESET_SESSION_NOT_FOUND", BadRequest, "Sesja resetowania hasła wygasła.")
	ErrInvalidResetCode     = newErr("INVALID_RESET_CODE", Validation, "Nieprawidłowy kod resetujący.")
	ErrUntrustedDevice      = newErr("UNTRUSTED_DEVICE", Unauthorized, "To urządzenie nie jest zaufane.")
	ErrInvalidSignature     = newErr("INVALID_SIGNATURE", Unauthorized, "Nieprawidłowy podpis bezpieczeństwa.")
)
