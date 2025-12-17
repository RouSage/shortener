package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/appvalidator"
)

// HTTPError represents an HTTP error response for swagger documentation
type HTTPError struct {
	Message string `json:"message" example:"error message"`
}

func (s *Server) failedValidationError(c echo.Context, err error) error {
	if appValidator, ok := c.Echo().Validator.(*appvalidator.AppValidator); ok {
		validationErrors := appValidator.FormatErrors(err)

		return c.JSON(http.StatusBadRequest, map[string]any{
			"message": "Validation failed",
			"errors":  validationErrors,
		})
	}

	return echo.ErrBadRequest
}
