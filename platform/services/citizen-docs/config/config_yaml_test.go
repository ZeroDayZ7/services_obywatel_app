package config

import (
	"os"
	"path/filepath"
	"testing"

	spfViper "github.com/spf13/viper"
)

func TestLoadConfigGlobalLoadsHMACConfigFromYaml(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	tmpDir := t.TempDir()
	configYAML := `
HMAC_TARGET_KEYS:
  gateway:
    target_key: yaml-gateway-docs
    algorithm: HmacSha256
  officer-bff:
    target_key: yaml-bff-docs
    algorithm: HmacSha256
HMAC_INTERNAL_KEYS:
  document_number:
    target_key: yaml-documents-number-index
    algorithm: HmacSha256
  docs-id-cards:
    target_key: yaml-docs-id-cards
    algorithm: AES256GCM
  docs-driver-license:
    target_key: yaml-docs-driver-license
    algorithm: AES256GCM
  docs-passport:
    target_key: yaml-docs-passport
    algorithm: AES256GCM
`

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configYAML), 0o600); err != nil {
		t.Fatalf("WriteFile config.yaml error = %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir(tmpDir) error = %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
		spfViper.Reset()
		_ = os.Unsetenv("KMS_SERVICE_NAME")
		_ = os.Unsetenv("KMS_SERVICE_SECRET")
		AppConfig = Config{}
	}()

	_ = os.Setenv("KMS_SERVICE_NAME", "citizen-docs")
	_ = os.Setenv("KMS_SERVICE_SECRET", "12345678901234567890123456789012")
	spfViper.Reset()
	AppConfig = Config{}

	if err := LoadConfigGlobal(); err != nil {
		t.Fatalf("LoadConfigGlobal() error = %v", err)
	}

	if got := AppConfig.HMAC.TargetKeys["gateway"].TargetKey; got != "yaml-gateway-docs" {
		t.Fatalf("gateway target key mismatch: got %q, want %q", got, "yaml-gateway-docs")
	}

	if got := AppConfig.HMAC.InternalKeys["document_number"].TargetKey; got != "yaml-documents-number-index" {
		t.Fatalf("document_number target key mismatch: got %q, want %q", got, "yaml-documents-number-index")
	}
}
