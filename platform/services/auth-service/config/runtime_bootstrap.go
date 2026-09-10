package config

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/zerodayz7/platform/pkg/agent"
	"github.com/zerodayz7/platform/pkg/viper"
)

var (
	ErrInvalidCredentials = errors.New("runtime credentials validation failed")
)

type RuntimeCredentials struct {
	resources map[string]any
}

func NewRuntimeCredentials(resp *agent.FullBootstrapResponse) *RuntimeCredentials {
	if resp == nil {
		return &RuntimeCredentials{resources: map[string]any{}}
	}

	resources := map[string]any{}
	if resp.Postgres != nil {
		resources["postgres"] = resp.Postgres
	}
	if resp.Redis != nil {
		resources["redis"] = resp.Redis
	}
	if resp.RabbitMQ != nil {
		resources["rabbitmq"] = resp.RabbitMQ
	}

	return &RuntimeCredentials{resources: resources}
}

func (r *RuntimeCredentials) Apply(handlers map[string]func(any) error) error {
	for name, handler := range handlers {
		value, ok := r.resources[name]
		if !ok || value == nil {
			continue
		}

		if err := handler(value); err != nil {
			return fmt.Errorf("apply bootstrap credentials for %s: %w", name, err)
		}
	}
	return nil
}

// sanitizeSecret ucina białe znaki, NUL-bajty (\x00) i znaki nowej linii (\r, \n) z bajtów haseł.
func sanitizeSecret(b []byte) string {
	cleanBytes := bytes.Trim(b, "\x00\r\n\t ")
	return strings.TrimSpace(string(cleanBytes))
}

// BuildRuntimeConfigs tworzy konfiguracje uruchomieniowe, wykonuje sanitację haseł oraz waliduje poprawność danych.
func BuildRuntimeConfigs(base Config, resp *agent.FullBootstrapResponse) (viper.DBConfig, viper.RedisConfig, viper.RabbitMQConfig, error) {
	dbCfg := base.Database
	redisCfg := base.Redis
	rabbitCfg := base.RabbitMQ

	runtimeCreds := NewRuntimeCredentials(resp)
	handlers := map[string]func(any) error{
		"postgres": func(value any) error {
			creds, ok := value.(*agent.PostgresCredentials)
			if !ok {
				return fmt.Errorf("invalid postgres credentials type %T", value)
			}

			user := strings.TrimSpace(creds.Username)
			pass := sanitizeSecret(creds.Password)

			if user == "" || pass == "" {
				return fmt.Errorf("%w: postgres username or password is empty after sanitization", ErrInvalidCredentials)
			}

			dbCfg.User = user
			dbCfg.Password = pass
			return nil
		},
		"redis": func(value any) error {
			creds, ok := value.(*agent.RedisCredentials)
			if !ok {
				return fmt.Errorf("invalid redis credentials type %T", value)
			}

			pass := sanitizeSecret(creds.Password)
			if pass == "" {
				return fmt.Errorf("%w: redis password is empty after sanitization", ErrInvalidCredentials)
			}

			if creds.Username != "" {
				redisCfg.Username = strings.TrimSpace(creds.Username)
			}
			redisCfg.Password = pass
			return nil
		},
		"rabbitmq": func(value any) error {
			creds, ok := value.(*agent.RabbitMQCredentials)
			if !ok {
				return fmt.Errorf("invalid rabbitmq credentials type %T", value)
			}

			user := strings.TrimSpace(creds.Username)
			pass := sanitizeSecret(creds.Password)

			if user == "" || pass == "" {
				return fmt.Errorf("%w: rabbitmq username or password is empty after sanitization", ErrInvalidCredentials)
			}

			rabbitCfg.User = user
			rabbitCfg.Password = pass
			return nil
		},
	}

	if err := runtimeCreds.Apply(handlers); err != nil {
		return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{}, err
	}

	return dbCfg, redisCfg, rabbitCfg, nil
}
