package constants

type TokenScope string

const (
	// ScopeDeviceVerify uprawnia wyłącznie do dokończenia procesu onboardingu urządzenia.
	ScopeDeviceVerify TokenScope = "device_verify"

	// ScopeAccess uprawnia do standardowego korzystania z chronionych endpointów API.
	ScopeAccess TokenScope = "access"

	// Scope2FAOnboarding opcjonalny scope dla procesu konfiguracji 2FA / Passkeys.
	Scope2FAOnboarding TokenScope = "2fa_onboarding"

	// ScopePasswordReset opcjonalny scope dla jednorazowych tokenów resetu hasła.
	ScopePasswordReset TokenScope = "password_reset"
)

//#region String
func (s TokenScope) String() string {
	return string(s)
}

// IsValid sprawdza, czy dany scope należy do dozwolonych w systemie.
//#region IsValid
func (s TokenScope) IsValid() bool {
	switch s {
	case ScopeDeviceVerify, ScopeAccess, Scope2FAOnboarding, ScopePasswordReset:
		return true
	default:
		return false
	}
}
