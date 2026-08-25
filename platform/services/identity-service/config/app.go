package config

// Load loads configuration and returns a `Config` instance.
// It delegates to the existing LoadConfigGlobal implementation for now.
func Load() (*Config, error) {
	if err := LoadConfigGlobal(); err != nil {
		return nil, err
	}

	return &AppConfig, nil
}