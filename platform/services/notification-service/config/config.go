package config

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

var AppConfig viper.Config

func LoadConfigGlobal() error {
	log := shared.GetLogger()
	if err := viper.InitConfig(&AppConfig, "notification-service"); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	log.Info("Configuration loaded and validated for notification-service")
	return nil
}
