package schemas

type RegisterRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=3,max=30"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,passwd"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password []byte `json:"password" validate:"required"`
}

type LoginStep2Request struct {
	UserID           string `json:"user_id" validate:"required"`
	CardSerialNumber string `json:"card_serial_number" validate:"required"`
	Challenge        string `json:"challenge" validate:"required"`
	Signature        string `json:"signature" validate:"required"`
}

type UnpairDeviceRequest struct {
	Signature string `json:"signature,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type VerifyDeviceRequest struct {
	Signature string `json:"signature" validate:"required"`
}

type TwoFARequest struct {
	Code  []byte `json:"code" validate:"required,numeric_byte,len=6"`
	Token string `json:"token" validate:"required"`
}

type ResendTwoFARequest struct {
	Email string `json:"email" validate:"required,email"`
	Token string `json:"token" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ===== Reset Password =====
type ResetPasswordRequest struct {
	AccountIdentifier string `json:"account_identifier" validate:"required"`
	Value             string `json:"value" validate:"required,email"`
	Method            string `json:"method" validate:"required,oneof=email sms"`
}

type ResetCodeVerifyRequest struct {
	Token string `json:"token" validate:"required"`
	Code  string `json:"code" validate:"required"`
}

type ResetPasswordFinalRequest struct {
	Token       string `json:"reset_token" validate:"required"`
	Code        string `json:"code" validate:"required,len=6"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
	Signature   string `json:"signature"`
	Fingerprint string `json:"fingerprint"`
	DeviceName  string `json:"device_name"`
	Platform    string `json:"platform"`
	PublicKey   string `json:"public_key"`
}

type RegisterDeviceRequest struct {
	PublicKey           string `json:"public_key" validate:"required"`
	Signature           string `json:"signature" validate:"required"`
	DeviceFingerprint   string `json:"fingerprint" validate:"required"`
	DeviceNameEncrypted string `json:"encrypted_name" validate:"required"`
	Platform            string `json:"platform" validate:"required"`
}

type FinalizeResetRequest struct {
	Token       string `json:"token" validate:"required"`
	Password    string `json:"password" validate:"required,min=8"`
	Signature   string `json:"signature" validate:"required"`
	Fingerprint string `json:"fingerprint" validate:"required"`
	PublicKey   string `json:"public_key" validate:"required"`
	DeviceName  string `json:"device_name" validate:"required"`
	Platform    string `json:"platform" validate:"required"`
}
