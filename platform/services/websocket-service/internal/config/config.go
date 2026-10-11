package config

import (
	"fmt"
	"time"

	spfViper "github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/shared"
	pkgViper "github.com/zerodayz7/platform/pkg/viper"
)

type Config struct {
	Server        pkgViper.ServerConfig `mapstructure:",squash"`
	Redis         pkgViper.RedisConfig  `mapstructure:",squash"`
	SharedSecret  string                `mapstructure:"SHARED_SECRET" validate:"required"`
	BroadcastKey  string                `mapstructure:"INTERNAL_BROADCAST_KEY" validate:"required"`
	Shutdown      time.Duration         `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

var AppConfig Config

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	pkgViper.SetBaseDefaults("websocket-service")
	pkgViper.SetRedisDefaults()

	spfViper.SetDefault("SHARED_SECRET", "dev-shared-secret")
	spfViper.SetDefault("INTERNAL_BROADCAST_KEY", "dev-internal-broadcast-key")

	if err := pkgViper.InitConfig(&AppConfig, "websocket-service"); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	log.Info("Configuration loaded and validated for websocket-service")
	return nil
}
