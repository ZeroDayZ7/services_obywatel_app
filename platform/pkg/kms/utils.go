package kms

import (
	"crypto/subtle"
)

func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
	subtle.ConstantTimeCompare(b, b)
}
