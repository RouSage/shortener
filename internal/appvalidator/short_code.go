package appvalidator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// Same symbols as go-nanoid supports
// See: https://github.com/matoous/go-nanoid?tab=readme-ov-file#go-nanoid
var shortCode = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func ValidateShortCode(fl validator.FieldLevel) bool {
	return shortCode.MatchString(fl.Field().String())
}
