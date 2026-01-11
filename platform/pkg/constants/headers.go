package constants

// HTTP Headers - Standard & Proxy
// Używane do poprawnej identyfikacji adresu IP oraz standardowej obsługi sesji.
const (
	HeaderXForwardedFor = "X-Forwarded-For"
	HeaderXRealIP       = "X-Real-IP"
	HeaderRequestID     = "X-Request-ID"
	HeaderAuth          = "Authorization"
	HeaderCookie        = "Cookie"
)

// HTTP Headers - User & Device Identification
// Kluczowe dla śledzenia kontekstu użytkownika i urządzenia wewnątrz systemu.
const (
	HeaderUserID            = "X-User-Id"
	HeaderSessionID         = "X-Session-Id"
	HeaderDeviceID          = "X-Device-Id"
	HeaderDeviceFingerprint = "X-Device-Fingerprint"
)

// HTTP Headers - Internal Security (Zero Trust)
// Wykorzystywane do bezpiecznej komunikacji między mikroserwisami.
const (
	// HeaderInternalContext zawiera zakodowany payload (base64) z danymi kontekstu.
	HeaderInternalContext = "X-Internal-Context"
	// HeaderInternalSignature służy do weryfikacji integralności payloadu.
	HeaderInternalSignature = "X-Internal-Signature"
)
