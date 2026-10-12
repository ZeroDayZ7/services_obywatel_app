package handler

import (
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	reqctx "github.com/zerodayz7/platform/pkg/context"
)

func TestTrustedUserIDFromSignedContext(t *testing.T) {
	secret := "dev-shared-secret"
	userID := uuid.MustParse("11111111-2222-3333-4444-555555666666")

	ctx := reqctx.RequestContext{UserID: &userID}
	payload, err := reqctx.Encode(ctx)
	if err != nil {
		t.Fatalf("encode request context: %v", err)
	}

	sig := reqctx.SignHMAC(payload, []byte(secret))
	encoded := base64.StdEncoding.EncodeToString(payload)

	got, err := trustedUserIDFromSignedContext(encoded, sig, secret, "attacker-user-id")
	if err != nil {
		t.Fatalf("expected valid signed context: %v", err)
	}
	if got != userID.String() {
		t.Fatalf("expected user id %s, got %s", userID.String(), got)
	}
}

func TestTrustedUserIDFromSignedContextRejectsMissingSignature(t *testing.T) {
	secret := "dev-shared-secret"
	_, err := trustedUserIDFromSignedContext("dGVzdA==", "", secret, "attacker-user-id")
	if err == nil {
		t.Fatal("expected error when payload or signature is missing")
	}
}
