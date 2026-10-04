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

func resolveOptionalSecret(provider secretprovider.SecretProvider, name string) (string, bool, error) {
	if provider == nil {
		return "", false, fmt.Errorf("secret provider initialization failed: provider is nil")
	}

	secret, err := provider.Get(context.Background(), name)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return "", false, nil
		}
		return "", false, fmt.Errorf("resolve secret %q: %w", name, err)
	}

	value := sanitizeSecret(secret)
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
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
		if rabbitUser, ok, err := resolveOptionalSecret(provider, secretprovider.RabbitMQUsernameSecret); err != nil {
			return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{}, fmt.Errorf("failed to resolve runtime credentials: %w", err)
		} else if ok {
			rabbitCfg.User = rabbitUser
		} else if rabbitCfg.User == "" {
			rabbitCfg.User = base.RabbitMQ.User
		}

		if rabbitPass, ok, err := resolveOptionalSecret(provider, secretprovider.RabbitMQPasswordSecret); err != nil {
			return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{}, fmt.Errorf("failed to resolve runtime credentials: %w", err)
		} else if ok {
			rabbitCfg.Password = rabbitPass
		} else if rabbitCfg.Password == "" {
			rabbitCfg.Password = base.RabbitMQ.Password
		}
	}

	return dbCfg, redisCfg, rabbitCfg, nil
}
