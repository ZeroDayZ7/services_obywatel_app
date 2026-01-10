package config

import (
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	pkgConfig "github.com/zerodayz7/platform/pkg/viper"
)

var (
	// AppConfig używa globalnego typu viper.Config dla pełnej spójności.
	AppConfig viper.Config
	Store     *session.Store
)

// LoadConfigGlobal ładuje konfigurację przy użyciu ustandaryzowanego InitConfig.
func LoadConfigGlobal() error {
	log := shared.GetLogger()

	// 1. Automatyczne mapowanie środowiska i pliku .env na strukturę AppConfig.
	if err := pkgConfig.InitConfig(&AppConfig, "audit-service"); err != nil {
		return err
	}

	log.Info("Audit-Service configuration loaded")
	return nil
}
