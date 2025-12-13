package middleware

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func init() {
	validate.RegisterValidation("passwd", func(fl validator.FieldLevel) bool {
		pw := fl.Field().String()

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

			if hasUpper && hasLower && hasDigit && hasSpecial {
				break
			}
		}

		return hasUpper && hasLower && hasDigit && hasSpecial
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

// ValidateStruct validates any struct using the global `validator` instance.
// By default, it returns a map of field names to validation tags (e.g., "required", "passwd").
// If you prefer human-readable error messages, you can use the commented-out version below,
// which maps validation tags to descriptive messages for each field.
// You can choose either approach depending on your use case.

//  var errorMessages = map[string]string{
// 	"required": "This field is required",
// 	"min":      "Minimum length not met",
// 	"alphanum": "Can only contain letters and numbers",
// 	"email":    "Invalid email address",
// 	"passwd":   "Password must be at least 8 chars, include uppercase, lowercase, number and special character",
// }

// func ValidateStruct(s any) map[string]string {
// 	errs := make(map[string]string)
// 	if err := validate.Struct(s); err != nil {
// 		for _, e := range err.(validator.ValidationErrors) {
// 			msg, ok := errorMessages[e.Tag()]
// 			if !ok {
// 				msg = e.Error()
// 			}
// 			errs[e.Field()] = msg
// 		}
// 	}
// 	if len(errs) > 0 {
// 		return errs
// 	}
// 	return nil
// }
