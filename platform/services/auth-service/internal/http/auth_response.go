package http

// LoginResponse defines the data returned after a successful or partial login attempt.
type LoginResponse struct {
	Type          string `json:"type,omitempty"`
	TwoFARequired bool   `json:"2fa_required"`
	TwoFAToken    string `json:"two_fa_token,omitempty"`
	AccessToken   string `json:"access_token,omitempty"`
	RefreshToken  string `json:"refresh_token,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	SetupToken    string `json:"setup_token,omitempty"`
	Challenge     string `json:"challenge,omitempty"`
	IsTrusted     bool   `json:"is_trusted,omitempty"`
	ExpiresAt     int64  `json:"expires_at,omitempty"`
}

// Verify2FAResponse defines the challenge and access data returned after successful 2FA verification.
type Verify2FAResponse struct {
	Success    bool   `json:"success"`
	SetupToken string `json:"setup_token"`
	Challenge  string `json:"challenge"`
	IsTrusted  bool   `json:"is_trusted"`
	UserID     string `json:"user_id"`
}

// RegisterDeviceResponse defines the outcome of a cryptographic device pairing process.
type RegisterDeviceResponse struct {
	Success      bool           `json:"success"`
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	IsTrusted    bool           `json:"is_trusted"`
	User         DeviceUserData `json:"user"`
	Rbac         map[string]any `json:"rbac"`
}

// DeviceUserData represents a subset of user information included in device registration.
type DeviceUserData struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Role        string   `json:"role"`
	LastLogin   string   `json:"last_login"`
	Roles       []string `json:"roles"`
}

// RefreshResponse defines the new security tokens issued during a refresh cycle.
type RefreshResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	UserID       string   `json:"user_id"`
	Roles        []string `json:"roles"`
	ExpiresAt    int64    `json:"expires_at"`
}

// LogoutResponse confirms the termination of the current user session and token revocation.
type LogoutResponse struct {
	Message string `json:"message"`
}

// RegisterResponse defines the data returned after successful creation of a new user account.
type RegisterResponse struct {
	Success bool `json:"success"`
}
