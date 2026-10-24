// cmdr: redis/constants.go

package constants

const (
	SessionPrefix          = "session:"       // Dla aktywnych sesji użytkowników
	UserSessionIndexPrefix = "user:sessions:" // Indeks aktywnych sesji przypisanych do użytkownika
	ChallengePrefix        = "challenge:"     // Dla wyzwań Ed25519
	Login2FAPrefix         = "login:2fa:"     // Dla tymczasowych sesji 2FA
	SetupSessionPrefix     = "setup:session:" // Dla rezerwacji urządzeń
)
