package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

// #region HashSHA256
func HashSHA256(data string) string {
	if data == "" {
		return ""
	}
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}
