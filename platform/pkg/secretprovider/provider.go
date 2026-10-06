package secretprovider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zerodayz7/platform/pkg/agent"
)

const (
	LocalProvider = "local"
	KMS12Provider = "kms12"

	PostgresUsernameSecret = "postgres.username"
	PostgresPasswordSecret = "postgres.password"
	RedisUsernameSecret    = "redis.username"
	RedisPasswordSecret    = "redis.password"
	RabbitMQUsernameSecret = "rabbitmq.username"
	RabbitMQPasswordSecret = "rabbitmq.password"
)

// SecretProvider resolves a named secret without exposing the source implementation.
type SecretProvider interface {
	Get(ctx context.Context, name string) ([]byte, error)
}

// Config describes provider-specific bootstrap requirements.
type Config struct {
	Provider      string
	ManifestPath  string
	SocketPath    string
	TargetService string
	Timeout       time.Duration
}

func New(cfg Config) (SecretProvider, error) {
	providerName := strings.TrimSpace(cfg.Provider)
	switch providerName {
	case "":
		return nil, fmt.Errorf("secret provider initialization failed: SECRET_PROVIDER is empty")
	case LocalProvider:
		return &DevSecretProvider{}, nil
	case KMS12Provider:
		return NewAgentSecretProvider(cfg)
	default:
		return nil, fmt.Errorf("unknown secret provider %q", providerName)
	}
}

// FakeSecretProvider is used by tests and callers wishing to inject a static secret map.
type FakeSecretProvider struct {
	Secrets map[string][]byte
}

func (f *FakeSecretProvider) Get(_ context.Context, name string) ([]byte, error) {
	if f == nil {
		return nil, fmt.Errorf("secret provider initialization failed: provider is nil")
	}
	if value, ok := f.Secrets[name]; ok {
		return append([]byte(nil), value...), nil
	}
	return nil, fmt.Errorf("secret %q not found", name)
}

// DevSecretProvider resolves secrets from the local environment for development/testing only.
type DevSecretProvider struct{}

func (p *DevSecretProvider) Get(_ context.Context, name string) ([]byte, error) {
	for _, envName := range envNamesForSecret(name) {
		value, ok := os.LookupEnv(envName)
		if !ok {
			continue
		}
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("secret %q is empty", name)
		}
		return []byte(value), nil
	}
	return nil, fmt.Errorf("secret %q not found", name)
}

func envNamesForSecret(name string) []string {
	switch name {
	case PostgresUsernameSecret:
		return []string{"AUTH_POSTGRES_USERNAME"}
	case PostgresPasswordSecret:
		return []string{"AUTH_POSTGRES_PASSWORD"}
	case RedisUsernameSecret:
		return []string{"AUTH_REDIS_USERNAME"}
	case RedisPasswordSecret:
		return []string{"AUTH_REDIS_PASSWORD"}
	case RabbitMQUsernameSecret:
		return []string{"AUTH_RABBITMQ_USERNAME"}
	case RabbitMQPasswordSecret:
		return []string{"AUTH_RABBITMQ_PASSWORD"}
	default:
		return nil
	}
}

// AgentSecretProvider adapts the existing Secret Agent bootstrap flow without exposing
// the sidecar implementation to the business code.
type AgentSecretProvider struct {
	mu           sync.RWMutex
	manifestPath string
	cfg          agent.Config
	response     *agent.FullBootstrapResponse
	cleanup      func()
}

func NewAgentSecretProvider(cfg Config) (*AgentSecretProvider, error) {
	manifestPath := strings.TrimSpace(cfg.ManifestPath)
	if manifestPath == "" {
		manifestPath = "secrets.yaml"
	}

	manifest, err := agent.LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("secret provider initialization failed: %w", err)
	}

	providerCfg := agent.Config{
		SocketPath:    strings.TrimSpace(cfg.SocketPath),
		TargetService: strings.TrimSpace(cfg.TargetService),
		Timeout:       cfg.Timeout,
	}
	if providerCfg.SocketPath == "" {
		providerCfg.SocketPath = manifest.SocketPath
	}
	if providerCfg.TargetService == "" {
		providerCfg.TargetService = manifest.Service
	}
	if providerCfg.Timeout == 0 {
		providerCfg.Timeout = manifest.Timeout
	}

	provider := &AgentSecretProvider{manifestPath: manifestPath, cfg: providerCfg}
	if err := provider.bootstrap(); err != nil {
		return nil, err
	}
	return provider, nil
}

func (p *AgentSecretProvider) bootstrap() error {
	manifest, err := agent.LoadManifest(p.manifestPath)
	if err != nil {
		return fmt.Errorf("secret provider initialization failed: %w", err)
	}

	requiredServices := manifest.GetEnabledResourceNames()
	resp, cleanup, err := agent.BootstrapApp(context.Background(), p.cfg, requiredServices)
	if err != nil {
		return fmt.Errorf("secret provider initialization failed: %w", err)
	}

	p.mu.Lock()
	p.response = resp
	p.cleanup = cleanup
	p.mu.Unlock()
	return nil
}

func (p *AgentSecretProvider) Get(_ context.Context, name string) ([]byte, error) {
	p.mu.RLock()
	response := p.response
	p.mu.RUnlock()
	if response == nil {
		return nil, fmt.Errorf("secret provider initialization failed: secret agent bootstrap not initialized")
	}

	switch name {
	case PostgresUsernameSecret:
		if response.Postgres == nil {
			return nil, fmt.Errorf("secret %q not found", name)
		}
		return []byte(response.Postgres.Username), nil
	case PostgresPasswordSecret:
		if response.Postgres == nil || len(response.Postgres.Password) == 0 {
			return nil, fmt.Errorf("secret %q not found", name)
		}
		return append([]byte(nil), response.Postgres.Password...), nil
	case RedisUsernameSecret:
		if response.Redis == nil {
			return nil, fmt.Errorf("secret %q not found", name)
		}
		return []byte(response.Redis.Username), nil
	case RedisPasswordSecret:
		if response.Redis == nil || len(response.Redis.Password) == 0 {
			return nil, fmt.Errorf("secret %q not found", name)
		}
		return append([]byte(nil), response.Redis.Password...), nil
	case RabbitMQUsernameSecret:
		if response.RabbitMQ == nil {
			return nil, fmt.Errorf("secret %q not found", name)
		}
		return []byte(response.RabbitMQ.Username), nil
	case RabbitMQPasswordSecret:
		if response.RabbitMQ == nil || len(response.RabbitMQ.Password) == 0 {
			return nil, fmt.Errorf("secret %q not found", name)
		}
		return append([]byte(nil), response.RabbitMQ.Password...), nil
	default:
		return nil, fmt.Errorf("secret %q not found", name)
	}
}

func (p *AgentSecretProvider) Cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cleanup != nil {
		p.cleanup()
		p.cleanup = nil
	}
	if p.response != nil {
		if p.response.Postgres != nil {
			clear(p.response.Postgres.Password)
		}
		if p.response.Redis != nil {
			clear(p.response.Redis.Password)
		}
		if p.response.RabbitMQ != nil {
			clear(p.response.RabbitMQ.Password)
		}
		p.response = nil
	}
}
