package config

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/agent"
	"github.com/zerodayz7/platform/pkg/viper"
)

// RuntimeCredentials keeps bootstrap credentials separate from static AppConfig values.
// The configuration object itself stays static, while runtime secrets are resolved only
// for the exact resources that need them during bootstrap and client initialization.
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

// BuildRuntimeConfigs materializes runtime-specific configuration from bootstrap response
// without mutating the statically loaded AppConfig values.
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
			dbCfg.User = creds.Username
			dbCfg.Password = string(creds.Password)
			return nil
		},
		"redis": func(value any) error {
			creds, ok := value.(*agent.RedisCredentials)
			if !ok {
				return fmt.Errorf("invalid redis credentials type %T", value)
			}
			if creds.Username != "" {
				redisCfg.Username = creds.Username
			}
			redisCfg.Password = string(creds.Password)
			return nil
		},
		"rabbitmq": func(value any) error {
			creds, ok := value.(*agent.RabbitMQCredentials)
			if !ok {
				return fmt.Errorf("invalid rabbitmq credentials type %T", value)
			}
			rabbitCfg.User = creds.Username
			rabbitCfg.Password = string(creds.Password)
			return nil
		},
	}

	if err := runtimeCreds.Apply(handlers); err != nil {
		return viper.DBConfig{}, viper.RedisConfig{}, viper.RabbitMQConfig{}, err
	}

	return dbCfg, redisCfg, rabbitCfg, nil
}
