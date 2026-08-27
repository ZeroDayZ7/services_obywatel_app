package http

type LoginResultType string

const (
	LoginResult2FARequired      LoginResultType = "2FA_REQUIRED"
	LoginResultSuccess          LoginResultType = "SUCCESS"
	LoginResultTemporarySuccess LoginResultType = "TEMPORARY_SUCCESS"
	LoginResultPreTrust         LoginResultType = "PRE_TRUST"
	LoginResultEmployeeTrust    LoginResultType = "EMPLOYEE_TRUST"
)

type Login2FAData struct {
	TwoFARequired bool   `json:"two_fa_required"`
	TwoFAToken    string `json:"two_fa_token"`
}

type LoginSuccessData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	ExpiresAt    int64  `json:"expires_at"`
}

type LoginPreTrustData struct {
	SetupToken string `json:"setup_token"`
	Challenge  string `json:"challenge"`
}

type LoginEmployeeTrustData struct {
	Challenge  string `json:"challenge"`
	SetupToken string `json:"setup_token"`
}

type LoginResponse struct {
	Type          LoginResultType         `json:"type"`
	TwoFA         *Login2FAData           `json:"two_fa,omitempty"`
	Success       *LoginSuccessData       `json:"success,omitempty"`
	PreTrust      *LoginPreTrustData      `json:"pre_trust,omitempty"`
	EmployeeTrust *LoginEmployeeTrustData `json:"employee_trust,omitempty"`
}
