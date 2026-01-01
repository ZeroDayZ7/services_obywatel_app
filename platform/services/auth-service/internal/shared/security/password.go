package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	memory      = 64 * 1024
	iterations  = 3
	parallelism = 2
	saltLength  = 16
	keyLength   = 32
)

// HashPassword generates a salted Argon2id hash of the password
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)
	return encodedSalt + "$" + encodedHash, nil
}

// VerifyPassword compares password with encoded Argon2id hash
// VerifyPassword compares password bytes with encoded Argon2id hash
// ZMIANA: password to teraz []byte
func VerifyPassword(password []byte, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 2 {
		return false, errors.New("incorrect password format in the database")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false, err
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false, err
	}

	computedHash := argon2.IDKey(password, salt, iterations, memory, parallelism, keyLength)

	// Constant-time comparison chroni przed timing attacks
	isValid := subtle.ConstantTimeCompare(hash, computedHash) == 1

	for i := range computedHash {
		computedHash[i] = 0
	}

	return isValid, nil
}
