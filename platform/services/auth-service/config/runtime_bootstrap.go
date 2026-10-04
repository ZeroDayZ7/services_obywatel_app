// cmdr: config\runtime_bootstrap.go

package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zerodayz7/platform/pkg/secretprovider"
	"github.com/zerodayz7/platform/pkg/viper"
)

var (
	ErrInvalidCredentials = errors.New("runtime credentials validation failed")
)

// sanitizeSecret removes whitespace and NUL/newline bytes from sensitive material.
func sanitizeSecret(b []byte) string {
	cleanBytes := bytes.Trim(b, "\x00\r\n\t ")
	return strings.TrimSpace(string(cleanBytes))
}

func resolveRequiredSecret(provider secretprovider.SecretProvider, name string) ([]byte, error) {
	if provider == nil {
		return nil, fmt.Errorf("secret provider initialization failed: provider is nil")
	}

	secret, err := provider.Get(context.Background(), name)
	if err != nil {
		return nil, fmt.Errorf("resolve secret %q: %w", name, err)
	}
	if len(secret) == 0 {
		return nil, fmt.Errorf("secret %q is empty", name)
	}
	return secret, nil
}

// BuildRuntimeConfigs resolves the credentials required to initialize runtime PostgreSQL and Redis clients.
func BuildRuntimeConfigs(base Config, provider secretprovider.SecretProvider) (viper.DBConfig, viper.RedisConfig, viper.RabbitMQConfig, error) {
	dbCfg := base.Database
	redisCfg := base.Redis
	rabbitCfg := base.RabbitMQ

	if provider == nil {
		return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{},
			fmt.Errorf("secret provider initialization failed: provider is nil")
	}

	postgresUser, err := resolveRequiredSecret(provider, secretprovider.PostgresUsernameSecret)
	if err != nil {
		return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{}, fmt.Errorf("failed to resolve runtime credentials: %w", err)
	}
	postgresPass, err := resolveRequiredSecret(provider, secretprovider.PostgresPasswordSecret)
	if err != nil {
		return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{}, fmt.Errorf("failed to resolve runtime credentials: %w", err)
	}
	user := strings.TrimSpace(string(postgresUser))
	pass := sanitizeSecret(postgresPass)
	if user == "" || pass == "" {
		return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{},
			fmt.Errorf("%w: postgres username or password is empty after sanitization", ErrInvalidCredentials)
	}
	dbCfg.User = user
	dbCfg.Password = pass

	redisPass, err := resolveRequiredSecret(provider, secretprovider.RedisPasswordSecret)
	if err != nil {
		return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{}, fmt.Errorf("failed to resolve runtime credentials: %w", err)
	}
	redisPassSanitized := sanitizeSecret(redisPass)
	if redisPassSanitized == "" {
		return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{},
			fmt.Errorf("%w: redis password is empty after sanitization", ErrInvalidCredentials)
	}
	if redisUser, err := provider.Get(context.Background(), secretprovider.RedisUsernameSecret); err == nil {
		if trimmed := strings.TrimSpace(string(redisUser)); trimmed != "" {
			redisCfg.Username = trimmed
		}
	}
	redisCfg.Password = redisPassSanitized

	if base.RabbitMQ.Enabled {
		rabbitUser, err := resolveRequiredSecret(provider, secretprovider.RabbitMQUsernameSecret)
		if err != nil {
			return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{}, fmt.Errorf("failed to resolve runtime credentials: %w", err)
		}
		rabbitPass, err := resolveRequiredSecret(provider, secretprovider.RabbitMQPasswordSecret)
		if err != nil {
			return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{}, fmt.Errorf("failed to resolve runtime credentials: %w", err)
		}
		rabbitUserTrimmed := strings.TrimSpace(string(rabbitUser))
		rabbitPassTrimmed := sanitizeSecret(rabbitPass)
		if rabbitUserTrimmed == "" || rabbitPassTrimmed == "" {
			return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{},
				fmt.Errorf("%w: rabbitmq username or password is empty after sanitization", ErrInvalidCredentials)
		}
		rabbitCfg.User = rabbitUserTrimmed
		rabbitCfg.Password = rabbitPassTrimmed
	}

	return dbCfg, redisCfg, rabbitCfg, nil
}
