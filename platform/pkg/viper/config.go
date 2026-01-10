package viper

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// Inicjalizacja walidatora
var validate = validator.New()

func InitConfig(cfg any, serviceName string) error {
	// 1. Domyślne wartości specyficzne dla serwisu
	SetSharedDefaults(serviceName)

	// 2. Konfiguracja Vipera
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 3. Odczyt pliku
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("błąd podczas czytania pliku config: %v", err)
		}
	}

	// 4. Mapowanie z DecodeHook (obsługa time.Duration i slice'ów)
	if err := viper.Unmarshal(cfg, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	))); err != nil {
		return fmt.Errorf("nie udało się zmapować konfiguracji na strukturę: %v", err)
	}

	// 5. Walidacja struktury
	if err := validate.Struct(cfg); err != nil {
		// Mapujemy błędy na czytelny format
		var errorMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errorMsgs = append(errorMsgs, fmt.Sprintf("- Pole '%s' nie spełnia warunku '%s' (wartość: %v)", err.Field(), err.Tag(), err.Value()))
		}
		return fmt.Errorf("walidacja konfiguracji nie powiodła się:\n%s", strings.Join(errorMsgs, "\n"))
	}

	return nil
}
