package config

import (
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	pkgConfig "github.com/zerodayz7/platform/pkg/viper"
)

var AppConfig viper.Config

// LoadConfigGlobal
func LoadConfigGlobal() error {
	log := shared.GetLogger()

	if err := pkgConfig.InitConfig(&AppConfig, "audit-service"); err != nil {
		return err
	}

	log.Info("Audit-Service configuration loaded")
	return nil
}
