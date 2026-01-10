package config

import (
	"fmt"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	pkgConfig "github.com/zerodayz7/platform/pkg/viper"
)

var (
	AppConfig viper.Config
	Store     *session.Store
)

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	// InitConfig automatycznie wypełni AppConfig danymi z .env
	// Używamy nazwy "citizen-docs" dla spójności logów/konfiguracji
	if err := pkgConfig.InitConfig(&AppConfig, "citizen-docs"); err != nil {
		return err
	}

	// Walidacja krytycznych pól dla serwisu dokumentów
	// if AppConfig.Internal.EncryptionKey == "" {
	// 	return fmt.Errorf("INTERNAL_ENCRYPTION_KEY is required for document security")
	// }

	if AppConfig.Internal.HashSalt == "" {
		return fmt.Errorf("INTERNAL_HASH_SALT is required for PESEL hashing")
	}

	log.Info("Citizen-Docs configuration loaded successfully")
	return nil
}
