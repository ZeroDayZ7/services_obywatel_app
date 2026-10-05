package service

import (
	"testing"

	"github.com/zerodayz7/platform/pkg/crypto"
)

func TestComputeDocumentNumberHash_UsesHMACSHA256(t *testing.T) {
	secret := []byte("document-number-secret")
	number := "ABC-123456"

	got := computeDocumentNumberHash(number, secret)
	want := crypto.ComputeHMAC256Hex([]byte(number), secret)

	if got != want {
		t.Fatalf("computeDocumentNumberHash() = %q, want %q", got, want)
	}
}
