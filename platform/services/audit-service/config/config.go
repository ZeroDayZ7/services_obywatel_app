package config

import (
	"fmt"
	"time"

	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type Config struct {
	Server   viper.ServerConfig           `mapstructure:",squash"`
	Database viper.DBConfig               `mapstructure:",squash"`
	Redis    viper.RedisConfig            `mapstructure:",squash"`
	RabbitMQ viper.RabbitMQConfig         `mapstructure:",squash"`
	Internal viper.InternalSecurityConfig `mapstructure:",squash"`
	OTEL     viper.OTELConfig             `mapstructure:",squash"`
	Shutdown time.Duration                `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

var AppConfig Config

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetBaseDefaults("audit-service")
	viper.SetDBDefaults()
	viper.SetRedisDefaults()

	if err := viper.InitConfig(&AppConfig, "audit-service"); err != nil {
		return fmt.Errorf("failed to initialize audit-service config: %w", err)
	}

	log.Info("Audit-Service configuration loaded")
	return nil
}