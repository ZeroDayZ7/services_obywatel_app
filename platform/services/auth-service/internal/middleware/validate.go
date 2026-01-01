package middleware

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/zerodayz7/platform/pkg/shared" // Importujemy Twój pakiet shared
)

var validate = validator.New()

func init() {
	validate.RegisterValidation("passwd", func(fl validator.FieldLevel) bool {
		log := shared.GetLogger() // Pobieramy Twój logger
		field := fl.Field()
		var passwordBytes []byte

		// Logujemy typ pola, który walidator próbuje sprawdzić
		log.DebugMap("Validator: checking field", map[string]any{
			"kind": field.Kind().String(),
			"type": field.Type().String(),
		})

		switch field.Kind() {
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.Uint8 {
				passwordBytes = field.Bytes()
			}
		case reflect.String:
			passwordBytes = []byte(field.String())
		default:
			log.Warn("Validator: unsupported field kind")
			return false
		}

		// LOG: Długość po parsowaniu
		passLen := len(passwordBytes)
		if passLen < 8 {
			log.WarnObj("Validator: password too short", map[string]any{"length": passLen})
			return false
		}

		var hasUpper, hasLower, hasDigit, hasSpecial bool

		for _, b := range passwordBytes {
			c := rune(b)
			switch {
			case 'A' <= c && c <= 'Z':
				hasUpper = true
			case 'a' <= c && c <= 'z':
				hasLower = true
			case '0' <= c && c <= '9':
				hasDigit = true
			case strings.ContainsRune("!@#$%^&*()-_=+[]{}|;:,.<>/?~`", c):
				hasSpecial = true
			}
		}

		// LOG: Wyniki skanowania znaków
		log.DebugMap("Validator: password flags", map[string]any{
			"upper":   hasUpper,
			"lower":   hasLower,
			"digit":   hasDigit,
			"special": hasSpecial,
		})

		isValid := hasUpper && hasLower && hasDigit && hasSpecial
		if !isValid {
			log.Warn("Validator: password missing required character types")
		}

		return isValid
	})
}

func ValidateStruct(s any) map[string]string {
	errs := make(map[string]string)
	if err := validate.Struct(s); err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			errs[e.Field()] = e.Tag()
		}
	}
	return errs
}
