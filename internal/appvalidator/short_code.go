package appvalidator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var shortCode = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func ValidateShortCode(fl validator.FieldLevel) bool {
	return shortCode.MatchString(fl.Field().String())
}
