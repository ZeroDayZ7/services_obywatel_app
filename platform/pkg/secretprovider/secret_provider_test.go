package secretprovider

import (
	"context"
	"os"
	"testing"
)

func TestDevSecretProvider_GetExisting(t *testing.T) {
	t.Setenv("AUTH_POSTGRES_PASSWORD", "dev-pass")
	provider := &DevSecretProvider{}

	secret, err := provider.Get(context.Background(), "postgres.password")
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if string(secret) != "dev-pass" {
		t.Fatalf("Get() = %q, want %q", string(secret), "dev-pass")
	}
}

func TestDevSecretProvider_MissingSecret(t *testing.T) {
	provider := &DevSecretProvider{}
	if _, err := provider.Get(context.Background(), "postgres.password"); err == nil {
		t.Fatal("Get() expected error for missing secret")
	}
}

func TestDevSecretProvider_EmptySecret(t *testing.T) {
	t.Setenv("AUTH_POSTGRES_PASSWORD", "")
	provider := &DevSecretProvider{}
	if _, err := provider.Get(context.Background(), "postgres.password"); err == nil {
		t.Fatal("Get() expected error for empty secret")
	}
}

func TestNewSecretProvider_Local(t *testing.T) {
	provider, err := New(Config{Provider: LocalProvider})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("New() returned nil provider")
	}
}

func TestNewSecretProvider_KMS12(t *testing.T) {
	if _, err := os.Stat("../../services/auth-service/secrets.yaml"); err != nil {
		t.Skip("Secret Agent manifest not available in this environment")
	}
	if _, err := os.Stat("/var/run/agent-sockets/agent.sock"); err != nil {
		t.Skip("Secret Agent socket not available in this environment")
	}
	provider, err := New(Config{Provider: KMS12Provider, ManifestPath: "../../services/auth-service/secrets.yaml", TargetService: "auth_service"})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("New() returned nil provider")
	}
}

func TestNewSecretProvider_UnknownProvider(t *testing.T) {
	if _, err := New(Config{Provider: "unknown"}); err == nil {
		t.Fatal("New() expected error for unknown provider")
	}
}

func TestNewSecretProvider_EmptyProvider(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("New() expected error for empty provider")
	}
}
