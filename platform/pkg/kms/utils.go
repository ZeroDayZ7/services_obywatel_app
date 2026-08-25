// cmdr: kms\utils.go

package kms

import (
	"crypto/subtle"
)

//#region ZeroBytes
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
	subtle.ConstantTimeCompare(b, b)
}
