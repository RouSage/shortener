package appvalidator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type AppValidator struct {
	validate *validator.Validate
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func New() *AppValidator {
	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		// skip if tag key says it should be ignored
		if name == "-" {
			return ""
		}
		return name
	})
	err := validate.RegisterValidation("shortcode", ValidateShortCode)
	if err != nil {
		panic(fmt.Errorf("register shortcode validator: %w", err))
	}

	return &AppValidator{
		validate: validate,
	}
}

func (av *AppValidator) Validate(i any) error {
	if err := av.validate.Struct(i); err != nil {
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
	case "shortcode":
		return "Short code cannot contain special characters"
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}
