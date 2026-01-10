package context

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

func Sign(payload []byte, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write(payload)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func Verify(payload []byte, signature string, secret []byte) bool {
	expected := Sign(payload, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}
