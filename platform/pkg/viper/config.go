package viper

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

var validate = validator.New()

func InitConfig(cfg any, serviceName string) error {
	SetSharedDefaults(serviceName)

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := bindEnvs(cfg); err != nil {
		return fmt.Errorf("failed to bind environment variables: %w", err)
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("błąd podczas czytania pliku config: %v", err)
		}
	}

	decodeHook := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	))

	if err := viper.Unmarshal(cfg, decodeHook); err != nil {
		return fmt.Errorf("nie udało się zmapować konfiguracji: %v", err)
	}

	if err := validate.Struct(cfg); err != nil {
		var errorMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errorMsgs = append(errorMsgs, fmt.Sprintf("- Pole '%s' nie spełnia warunku '%s' (wartość: %v)", err.Field(), err.Tag(), err.Value()))
		}
		return fmt.Errorf("walidacja konfiguracji nie powiodła się:\n%s", strings.Join(errorMsgs, "\n"))
	}

	return nil
}

func bindEnvs(cfg any) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Type.Kind() == reflect.Struct {
			if err := bindEnvs(v.Field(i).Addr().Interface()); err != nil {
				return err
			}
			continue
		}

		tag := field.Tag.Get("mapstructure")
		if tag != "" && tag != ",squash" {
			if err := viper.BindEnv(tag); err != nil {
				return fmt.Errorf("error binding env var %s: %w", tag, err)
			}
		}
	}
	return nil
}
