package agent

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type ResourceSpec struct {
	Name             string        `yaml:"name"`
	Type             string        `yaml:"type"`
	RotationInterval time.Duration `yaml:"rotation_interval"`
	Enabled          bool          `yaml:"enabled"`
}

type Manifest struct {
	Service     string         `yaml:"service"`
	SocketPath  string         `yaml:"socket_path"`
	Timeout     time.Duration  `yaml:"timeout"`
	Credentials []ResourceSpec `yaml:"credentials"`
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu manifestu agenta [%s]: %w", path, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("błąd unmarshal manifestu agenta: %w", err)
	}

	// Nadpisujemy z ENV jeśli zmienna istnieje w środowisku
	if envSocket := os.Getenv("SECRET_AGENT_SOCKET_PATH"); envSocket != "" {
		m.SocketPath = envSocket
	}

	if m.Timeout == 0 {
		m.Timeout = 5 * time.Second
	}

	return &m, nil
}

// GetEnabledResourceNames zwraca listę nazw zasobów, które mają flaga enabled = true.
func (m *Manifest) GetEnabledResourceNames() []string {
	names := make([]string, 0, len(m.Credentials))
	for _, spec := range m.Credentials {
		if spec.Enabled {
			names = append(names, spec.Name)
		}
	}
	return names
}
