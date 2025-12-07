package auth

import (
	"context"
	"slices"
	"strings"

	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/labstack/echo/v4"
)

type contextKey string

const (
	ClaimsContextKey contextKey = "claims"
)

// CustomClaims contains custom data we want from the token
type CustomClaims struct {
	Scope string `json:"scope"`
}

// Validate does nothing, but we need it to satisfy validator.CustomClaims interface
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// HasScope checks whether claims have a specific scope
func (c CustomClaims) HasScope(expectedScope string) bool {
	result := strings.Split(c.Scope, " ")
	return slices.Contains(result, expectedScope)
}

func GetUserID(c echo.Context) *string {
	claims := getClaimsFromContext(c)
	if claims == nil || claims.RegisteredClaims.Subject == "" {
		return nil
	}

	return &claims.RegisteredClaims.Subject
}

func getClaimsFromContext(c echo.Context) *validator.ValidatedClaims {
	claims, ok := c.Get(string(ClaimsContextKey)).(*validator.ValidatedClaims)
	if !ok || claims == nil {
		return nil
	}

	return claims
}
