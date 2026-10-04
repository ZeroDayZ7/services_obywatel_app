package config

import (
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
