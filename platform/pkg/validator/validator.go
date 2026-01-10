package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Mapa komunikatów błędów - Single Source of Truth
var errorMessages = map[string]string{
	"required":     "This field is required",
	"min":          "Minimum length not met",
	"max":          "Maximum length exceeded",
	"len":          "Must be exactly 6 characters",
	"alphanum":     "Can only contain letters and numbers",
	"email":        "Invalid email address",
	"passwd":       "Password must be at least 8 chars, include uppercase, lowercase, number and special character",
	"numeric_byte": "This field must contain only digits",
}

func init() {
	if err := validate.RegisterValidation("passwd", validatePassword); err != nil {
		panic("failed to register passwd validation: " + err.Error())
	}

	// REJESTRACJA WALIDATORA CYFR (To rozwiąże błąd unusedfunc)
	if err := validate.RegisterValidation("numeric_byte", validateNumericByte); err != nil {
		panic("failed to register numeric_byte validation: " + err.Error())
	}
}

// Walidator dla []byte lub string - sprawdza czy są same cyfry
func validateNumericByte(fl validator.FieldLevel) bool {
	field := fl.Field()
	var val []byte

	if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
		val = field.Bytes()
	} else if field.Kind() == reflect.String {
		val = []byte(field.String())
	}

	if len(val) == 0 {
		return false
	}

	for _, b := range val {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

func validatePassword(fl validator.FieldLevel) bool {
	field := fl.Field()
	var pw string

	// Sprawdzamy, czy to []byte (Slice) czy string
	if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
		pw = string(field.Bytes())
	} else {
		pw = field.String()
	}

	if len(pw) < 8 {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range pw {
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
	return hasUpper && hasLower && hasDigit && hasSpecial
}

func Validate(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	errs := make(map[string]string)
	for _, e := range err.(validator.ValidationErrors) {
		msg, ok := errorMessages[e.Tag()]
		if !ok {
			msg = "Invalid value: " + e.Tag()
		}
		errs[e.Field()] = msg
	}
	return errs
}
