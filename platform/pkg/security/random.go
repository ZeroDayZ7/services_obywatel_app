package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
)

// GenerateRandomBytes generuje kryptograficznie bezpieczne losowe bajty
func GenerateRandomBytes(length int) ([]byte, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("security: failed to generate random bytes: %w", err)
	}
	return b, nil
}

// GenerateRandomString generuje URL-safe token (używane dla Refresh Tokenów i Challenge)
func GenerateRandomString(length int) (string, error) {
	b, err := GenerateRandomBytes(length)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateRefreshToken generuje domyślny 64-bajtowy token odświeżający
func GenerateRefreshToken() (string, error) {
	return GenerateRandomString(64)
}

// GenerateOTP generuje cyfrowy kod jednorazowy o określonej długości (np. 6 cyfr: "012345")
func GenerateOTP(digits int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("security: failed to generate OTP: %w", err)
	}

	format := fmt.Sprintf("%%0%dd", digits)
	return fmt.Sprintf(format, n.Int64()), nil
}
