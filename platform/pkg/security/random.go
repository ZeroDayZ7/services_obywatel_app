// cmdr: security\random.go

package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"

	"github.com/zerodayz7/platform/pkg/kms"
)

// GenerateRandomBytes generuje kryptograficznie bezpieczne losowe bajty
// #region GenerateRandomBytes
func GenerateRandomBytes(length int) ([]byte, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("security: failed to generate random bytes: %w", err)
	}
	return b, nil
}

// GenerateRandomString generuje URL-safe token (używane dla Refresh Tokenów i Challenge)
// #region GenerateRandomString
func GenerateRandomString(length int) (string, error) {
	b, err := GenerateRandomBytes(length)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// GenerateRefreshToken generuje domyślny 64-bajtowy token odświeżający
// #region GenerateRefreshToken
func GenerateRefreshToken() (string, error) {
	return GenerateRandomString(47)
}

// GenerateOTPBytes generuje cyfrowy kod jednorazowy o określonej długości jako []byte.
// Unika alokacji stringów w pamięci RAM.
// #region GenerateOTPBytes
func GenerateOTPBytes(digits int) ([]byte, error) {
	if digits <= 0 || digits > 10 {
		return nil, errors.New("security: invalid OTP digits count")
	}

	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, fmt.Errorf("security: failed to generate OTP: %w", err)
	}

	buf := make([]byte, digits)
	val := n.Int64()

	// Wypełniamy bufor od końca cyframi w ASCII ('0' - '9') z wyzerowaniem wiodącym
	for i := digits - 1; i >= 0; i-- {
		buf[i] = byte('0' + (val % 10))
		val /= 10
	}

	return buf, nil
}

// GenerateOTPString to pomocnicza funkcja dla miejsc wymagających typu string (np. PUK/weryfikacja).
func GenerateOTPString(digits int) (string, error) {
	bytes, err := GenerateOTPBytes(digits)
	if err != nil {
		return "", err
	}
	defer kms.ZeroBytes(bytes)
	return string(bytes), nil
}

// HashOTP generuje szybki i bezpieczny hash SHA-256 z kodem OTP.
// #region HashOTP
func HashOTP(otpBytes []byte) string {
	h := sha256.New()
	h.Write(otpBytes)
	digest := h.Sum(nil)
	defer kms.ZeroBytes(digest)

	return base64.RawStdEncoding.EncodeToString(digest)
}
