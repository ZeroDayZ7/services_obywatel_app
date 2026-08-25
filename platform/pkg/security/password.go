package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var DefaultParams = Params{
	Memory:      64 * 1024, // 64 MB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// Granice bezpieczeństwa parametrów (zapobieganie DoS przy naruszonej bazie)
const (
	MinMemory      = 8 * 1024    // 8 MB
	MaxMemory      = 1024 * 1024 // 1 GB
	MinIterations  = 1
	MaxIterations  = 10
	MinParallelism = 1
	MaxParallelism = 16
)

var (
	ErrInvalidHash        = errors.New("invalid password hash format")
	ErrInsecureHashParams = errors.New("hash parameters are outside safe boundaries")
)

// HashPassword generuje hash Argon2id w formacie PHC ($argon2id$v=19$m=...,t=...,p=...$salt$hash).
// Opcjonalny pepper dodaje kolejną warstwę ochrony (trzymany np. w Vault / Envs, nie w DB).
//#region HashPassword
func HashPassword(password []byte, pepper []byte) (string, error) {
	params := DefaultParams

	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	pepperedPassword := pepperPassword(password, pepper)

	hash := argon2.IDKey(
		pepperedPassword,
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Czyszczenie tymczasowych buforów wewnętrznych
	clear(pepperedPassword)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	result := fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		params.Memory,
		params.Iterations,
		params.Parallelism,
		encodedSalt,
		encodedHash,
	)

	clear(hash)
	clear(salt)

	return result, nil
}

// VerifyPassword weryfikuje hasło z hashem przy zachowaniu Constant-Time Compare.
//#region VerifyPassword
func VerifyPassword(password []byte, encoded string, pepper []byte) (bool, error) {
	params, salt, hash, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}

	pepperedPassword := pepperPassword(password, pepper)

	computedHash := argon2.IDKey(
		pepperedPassword,
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	clear(pepperedPassword)

	valid := subtle.ConstantTimeCompare(hash, computedHash) == 1

	clear(computedHash)
	clear(salt)

	return valid, nil
}

// NeedsRehash sprawdza, czy hash wymaga aktualizacji ze względu na nieaktualne parametry.
//#region NeedsRehash
func NeedsRehash(encoded string) bool {
	params, _, _, err := decodeHash(encoded)
	if err != nil {
		return true
	}

	current := DefaultParams

	return params.Memory != current.Memory ||
		params.Iterations != current.Iterations ||
		params.Parallelism != current.Parallelism ||
		params.KeyLength != current.KeyLength
}

// decodeHash parsuje i waliduje ciąg PHC pod kątem bezpiecznych limitów zasobów.
//#region decodeHash
func decodeHash(encoded string) (Params, []byte, []byte, error) {
	var params Params

	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return params, nil, nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" || parts[2] != "v=19" {
		return params, nil, nil, ErrInvalidHash
	}

	values := strings.SplitSeq(parts[3], ",")
	for value := range values {
		item := strings.Split(value, "=")
		if len(item) != 2 {
			return params, nil, nil, ErrInvalidHash
		}

		// Parsujemy bezpośrednio jako uint32 (bitSize: 32)
		val, err := strconv.ParseUint(item[1], 10, 32)
		if err != nil {
			return params, nil, nil, ErrInvalidHash
		}

		number := uint32(val)

		switch item[0] {
		case "m":
			params.Memory = number
		case "t":
			params.Iterations = number
		case "p":
			if number > 255 {
				return params, nil, nil, ErrInvalidHash
			}
			params.Parallelism = uint8(number)
		}
	}

	// Walidacja granic bezpieczeństwa (przed przydzieleniem zasobów w IDKey)
	if params.Memory < MinMemory || params.Memory > MaxMemory ||
		params.Iterations < MinIterations || params.Iterations > MaxIterations ||
		params.Parallelism < MinParallelism || params.Parallelism > MaxParallelism {
		return params, nil, nil, ErrInsecureHashParams
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params, nil, nil, ErrInvalidHash
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return params, nil, nil, ErrInvalidHash
	}

	params.SaltLength = uint32(len(salt))
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}

// pepperPassword łączy hasło z pepperczykiem przy użyciu HMAC-SHA256
//#region pepperPassword
func pepperPassword(password []byte, pepper []byte) []byte {
	if len(pepper) == 0 {
		buf := make([]byte, len(password))
		copy(buf, password)
		return buf
	}

	mac := hmac.New(sha256.New, pepper)
	mac.Write(password)
	return mac.Sum(nil)
}
