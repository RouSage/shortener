package appvalidator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type AppValidator struct {
	validator *validator.Validate
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func New() *AppValidator {
	return &AppValidator{
		validator: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (av *AppValidator) Validate(i interface{}) error {
	if err := av.validator.Struct(i); err != nil {
		return err
	}

	return nil
}

func (av *AppValidator) FormatErrors(err error) []ValidationError {
	var validationErrors []ValidationError

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			validationErrors = append(validationErrors, ValidationError{
				Field:   av.getFieldName(e),
				Message: av.getErrorMessage(e),
			})
		}
	}

	return validationErrors
}

func (av *AppValidator) getFieldName(fe validator.FieldError) string {
	return strings.ToLower(fe.Field())
}

func (av *AppValidator) getErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", fe.Field(), fe.Param())
	case "http_url":
		return "Invalid URL format"
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", fe.Field(), fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", fe.Field(), fe.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", fe.Field(), fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", fe.Field(), fe.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}
