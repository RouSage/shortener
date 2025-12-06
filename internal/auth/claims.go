package auth

import (
	"context"
	"slices"
	"strings"
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
