package appvalidator

import "github.com/go-playground/validator/v10"

type AppValidator struct {
	validator *validator.Validate
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
