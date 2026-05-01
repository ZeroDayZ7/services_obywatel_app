package config

import (
	"log"

	"github.com/spf13/viper"
)

type VersionConfig struct {
	Port             string
	Min              string
	Latest           string
	Force            bool
	UpdateUrlWindows string
}

func Get() VersionConfig {

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Println("Warning: config file not found, using env variables only")
	}

	viper.SetDefault("VERSION_SERVICE_PORT", "3005")
	viper.SetDefault("MIN_VERSION", "0.0.0")
	viper.SetDefault("LATEST_VERSION", "0.0.0")
	viper.SetDefault("FORCE_UPDATE", false)
	viper.SetDefault("UPDATE_URL_WINDOWS", "")

	return VersionConfig{
		Port:             viper.GetString("VERSION_SERVICE_PORT"),
		Min:              viper.GetString("MIN_VERSION"),
		Latest:           viper.GetString("LATEST_VERSION"),
		Force:            viper.GetBool("FORCE_UPDATE"),
		UpdateUrlWindows: viper.GetString("UPDATE_URL_WINDOWS"),
	}
}
