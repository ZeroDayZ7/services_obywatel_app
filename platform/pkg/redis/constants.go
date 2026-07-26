// cmdr: redis/constants.go

package redis

const (
	SessionPrefix      = "session:"       // Dla aktywnych sesji użytkowników
	ChallengePrefix    = "challenge:"     // Dla wyzwań Ed25519
	Login2FAPrefix     = "login:2fa:"     // Dla tymczasowych sesji 2FA
	SetupSessionPrefix = "setup:session:" // Dla rezerwacji urządzeń
)
