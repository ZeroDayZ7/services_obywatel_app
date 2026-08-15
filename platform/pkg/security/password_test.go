package security_test

import (
	"strings"
	"testing"

	"github.com/zerodayz7/platform/pkg/security"
)

func TestHashAndVerifyPassword(t *testing.T) {
	password := []byte("SuperSecretPassword123!")
	pepper := []byte("server-side-secret-pepper")

	defer clear(password)

	hash, err := security.HashPassword(password, pepper)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Errorf("unexpected hash format: %s", hash)
	}

	// Poprawne hasło i pepper
	valid, err := security.VerifyPassword(password, hash, pepper)
	if err != nil || !valid {
		t.Errorf("expected password to be valid, got valid=%v, err=%v", valid, err)
	}

	// Błędne hasło
	invalid, err := security.VerifyPassword([]byte("WrongPassword"), hash, pepper)
	if err != nil || invalid {
		t.Errorf("expected password to be invalid, got valid=%v, err=%v", invalid, err)
	}

	// Błędny pepper
	invalidPepper, err := security.VerifyPassword(password, hash, []byte("wrong-pepper"))
	if err != nil || invalidPepper {
		t.Errorf("expected password to fail verification with wrong pepper")
	}
}

func TestNeedsRehash(t *testing.T) {
	password := []byte("SuperSecretPassword123!")
	pepper := []byte("pepper")

	hash, err := security.HashPassword(password, pepper)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if security.NeedsRehash(hash) {
		t.Errorf("expected NeedsRehash to be false for fresh hash")
	}

	// Sztucznie modyfikujemy parametry na starsze (m=32768)
	oldHash := strings.Replace(hash, "m=65536", "m=32768", 1)
	if !security.NeedsRehash(oldHash) {
		t.Errorf("expected NeedsRehash to be true for modified parameters")
	}
}

func TestDoSBoundaryLimits(t *testing.T) {
	// Próba podstawienia skrajnych parametrów (Memory > MaxMemory)
	maliciousHash := "$argon2id$v=19$m=2097152,t=3,p=2$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g"

	valid, err := security.VerifyPassword([]byte("pass"), maliciousHash, nil)
	if valid || err != security.ErrInsecureHashParams {
		t.Errorf("expected ErrInsecureHashParams, got valid=%v, err=%v", valid, err)
	}
}
