package viper

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// InitConfig ładuje konfigurację do przekazanego wskaźnika struktury
func InitConfig(cfg any, serviceName string) error {
	// 1. Domyślne wartości specyficzne dla serwisu
	SetSharedDefaults(serviceName)

	// 2. Konfiguracja Vipera
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	// Pozwala na czytanie zmiennych z kropkami jako podkreślniki (np. SERVER.PORT -> SERVER_PORT)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 3. Odczyt pliku
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %v", err)
		}
	}

	// 4. Magia: Mapowanie automatyczne na strukturę
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unable to decode into struct, %v", err)
	}

	return nil
}
