// cmdr: viper\config.go

package viper

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

var validate = validator.New()

//#region InitConfig
func InitConfig(cfg any, serviceName string) error {
	SetBaseDefaults(serviceName)

	// Sprawdź czy zdefiniowano środowisko (domyślnie dev/env)
	envFile := os.Getenv("APP_ENV")
	if envFile == "" {
		envFile = ".env"
	}

	viper.SetConfigName(envFile)
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./services/" + serviceName)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Włącz automatyczne czytanie zmiennych środowiskowych
	viper.AutomaticEnv()

	// Najpierw bindujemy zmienne z tagów
	if err := bindEnvs(cfg); err != nil {
		return fmt.Errorf("failed to bind environment variables: %w", err)
	}

	// Spróbuj odczytać plik, ale nie przerywaj jeśli go nie ma (bo zmienne mogą być z Dockera)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Zgłoś błąd tylko jeśli plik istnieje, ale ma błędny format
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) {
				return fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	decodeHook := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	))

	if err := viper.Unmarshal(cfg, decodeHook); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validate.Struct(cfg); err != nil {
		var errorMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errorMsgs = append(errorMsgs, fmt.Sprintf("- Field '%s' failed on '%s' (value: %v)", err.Field(), err.Tag(), err.Value()))
		}
		return fmt.Errorf("config validation failed:\n%s", strings.Join(errorMsgs, "\n"))
	}

	return nil
}

//#region bindEnvs
func bindEnvs(cfg any) error {
	v := reflect.ValueOf(cfg)

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		if field.Type.Kind() == reflect.Struct {
			if err := bindEnvs(fieldVal.Addr().Interface()); err != nil {
				return err
			}
			continue
		}

		tag := field.Tag.Get("mapstructure")

		if tag != "" && !strings.Contains(tag, "squash") {
			if err := viper.BindEnv(tag); err != nil {
				return fmt.Errorf("error binding env var %s: %w", tag, err)
			}
		}
	}
	return nil
}
