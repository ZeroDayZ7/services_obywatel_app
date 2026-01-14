// redis/constants.go
package redis

const (
	SessionPrefix      = "session:"   // Dla aktywnych sesji użytkowników
	ChallengePrefix    = "challenge:" // Dla wyzwań Ed25519 (krótki TTL)
	Login2FAPrefix     = "login:2fa:" // Dla tymczasowych sesji 2FA (kod 6-cyfrowy)
	SetupSessionPrefix = "setup:session:"
)
