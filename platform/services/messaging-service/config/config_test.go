package config

import (
	"os"
	"testing"
)

func TestLoadConfigGlobalDefaultsRabbitMQEnabledFalse(t *testing.T) {
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Chdir(..) error = %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	AppConfig = Config{}

	if err := LoadConfigGlobal(); err != nil {
		t.Fatalf("LoadConfigGlobal() error = %v", err)
	}

	if AppConfig.RabbitMQEnabled {
		t.Fatal("RABBITMQ_ENABLED should default to false")
	}
}
