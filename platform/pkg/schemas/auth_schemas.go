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

type TwoFARequest struct {
	Code  []byte `json:"code" validate:"required,numeric_byte,len=6"`
	Token string `json:"token" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ===== Reset Password =====
type ResetPasswordRequest struct {
	Value  string `json:"value" validate:"required,email"`
	Method string `json:"method"`
}

type ResetCodeVerifyRequest struct {
	Token string `json:"token" validate:"required"`
	Code  string `json:"code" validate:"required"`
}

type ResetPasswordFinalRequest struct {
	Token       string `json:"reset_token" validate:"required"`
	Code        string `json:"code" validate:"required,len=6"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
	Signature   string `json:"signature" validate:"required"`
	Fingerprint string `json:"fingerprint" validate:"required"`
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
