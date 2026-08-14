package constants

// HTTP Headers - Standard & Proxy
const (
	HeaderXForwardedFor = "X-Forwarded-For"
	HeaderXRealIP       = "X-Real-IP"
	HeaderRequestID     = "X-Request-ID"
	HeaderAuth          = "Authorization"
	HeaderCookie        = "Cookie"
	HeaderContentType   = "Content-Type"
	HeaderAccept        = "Accept"
	HeaderUserAgent     = "User-Agent"
)

// HTTP Headers - User & Device Identification
const (
	HeaderUserID            = "X-User-ID"
	HeaderSessionID         = "X-Session-ID"
	HeaderDeviceID          = "X-Device-ID"
	HeaderDeviceFingerprint = "X-Device-Fingerprint"
)

// HTTP Headers - Internal Security (Zero Trust)
const (
	HeaderInternalContext   = "X-Internal-Context"
	HeaderInternalSignature = "X-Internal-Signature"
)
