package config

import (
	"log"

	"github.com/spf13/viper"
)

type VersionConfig struct {
	Port   string
	Min    string
	Latest string
	Force  bool
}

func Get() VersionConfig {
	// Ustawienie pliku konfiguracyjnego
	viper.SetConfigFile(".env") // lub "config.yaml" jeśli wolisz YAML
	viper.AutomaticEnv()        // pozwala nadpisać zmienne z systemu

	// Wczytanie pliku
	if err := viper.ReadInConfig(); err != nil {
		log.Println("Warning: config file not found, using env variables only")
	}

	// Domyślne wartości
	viper.SetDefault("VERSION_SERVICE_PORT", "3005")
	viper.SetDefault("MIN_VERSION", "0.0.0")
	viper.SetDefault("LATEST_VERSION", "0.0.0")
	viper.SetDefault("FORCE_UPDATE", false)

	return VersionConfig{
		Port:   viper.GetString("VERSION_SERVICE_PORT"),
		Min:    viper.GetString("MIN_VERSION"),
		Latest: viper.GetString("LATEST_VERSION"),
		Force:  viper.GetBool("FORCE_UPDATE"),
	}
}
