package config

import (
	"strings"
	"testing"

	"github.com/zerodayz7/platform/pkg/secretprovider"
	"github.com/zerodayz7/platform/pkg/viper"
)

func TestBuildRuntimeConfigsUsesSecretProvider(t *testing.T) {
	provider := &secretprovider.FakeSecretProvider{Secrets: map[string][]byte{
		secretprovider.PostgresUsernameSecret: []byte("auth_user"),
		secretprovider.PostgresPasswordSecret: []byte("local-pass"),
		secretprovider.RedisUsernameSecret:    []byte("redis_root"),
		secretprovider.RedisPasswordSecret:    []byte("redis-pass"),
	}}

	base := Config{
		Database: viper.DBConfig{Host: "127.0.0.1", Port: 5432, DBName: "auth_database", SSLMode: "disable"},
		Redis:    viper.RedisConfig{Host: "127.0.0.1", Port: "6379", DB: 0},
	}

	db, redis, _, err := BuildRuntimeConfigs(base, provider)
	if err != nil {
		t.Fatalf("BuildRuntimeConfigs() error = %v", err)
	}
	if db.User != "auth_user" || db.Password != "local-pass" {
		t.Fatalf("database creds not resolved from provider: %+v", db)
	}
	if redis.Username != "redis_root" || redis.Password != "redis-pass" {
		t.Fatalf("redis creds not resolved from provider: %+v", redis)
	}
}

func TestBuildRuntimeConfigsSkipsMissingRabbitMQSecrets(t *testing.T) {
	provider := &secretprovider.FakeSecretProvider{Secrets: map[string][]byte{
		secretprovider.PostgresUsernameSecret: []byte("auth_user"),
		secretprovider.PostgresPasswordSecret: []byte("local-pass"),
		secretprovider.RedisUsernameSecret:    []byte("redis_root"),
		secretprovider.RedisPasswordSecret:    []byte("redis-pass"),
	}}

	base := Config{
		Database: viper.DBConfig{Host: "127.0.0.1", Port: 5432, DBName: "auth_database", SSLMode: "disable"},
		Redis:    viper.RedisConfig{Host: "127.0.0.1", Port: "6379", DB: 0},
		RabbitMQ: viper.RabbitMQConfig{Enabled: true, Host: "localhost", Port: 5672, User: "fallback-user", Password: "fallback-pass"},
	}

	_, _, rabbit, err := BuildRuntimeConfigs(base, provider)
	if err != nil {
		t.Fatalf("BuildRuntimeConfigs() error = %v", err)
	}
	if rabbit.User != "fallback-user" || rabbit.Password != "fallback-pass" {
		t.Fatalf("missing RabbitMQ secret should not block startup: %+v", rabbit)
	}
}

func TestLoadConfigGlobalDefaultsSecretProviderToLocal(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_NAME", "auth-service")
	t.Setenv("PORT", "8082")
	t.Setenv("AUTH_DOMAIN", "obywatel.gov.pl")
	t.Setenv("JWT_ACCESS_TTL", "15m")
	t.Setenv("JWT_REFRESH_TTL", "168h")
	t.Setenv("SESSION_TTL", "60m")
	t.Setenv("KMS_ENDPOINT", "http://127.0.0.1:8080")
	t.Setenv("KMS_SERVICE_NAME", "auth-service")
	t.Setenv("KMS_SERVICE_SECRET", "super-long-random-secret-for-auth-service-hmac-64-bytes")
	t.Setenv("KMS_TIMEOUT", "2s")
	t.Setenv("SECRET_PROVIDER", "")

	if err := LoadConfigGlobal(); err != nil {
		t.Fatalf("LoadConfigGlobal() error = %v", err)
	}
	if got := strings.TrimSpace(AppConfig.Secret.Provider); got == "" {
		t.Fatal("SECRET_PROVIDER default must be set to local when env is empty")
	}
	if got := strings.TrimSpace(AppConfig.Secret.Provider); got != secretprovider.LocalProvider {
		t.Fatalf("SECRET_PROVIDER = %q, want %q", got, secretprovider.LocalProvider)
	}
}
