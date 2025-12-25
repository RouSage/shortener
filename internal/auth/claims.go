package auth

import (
	"context"
	"slices"
	"strings"

	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
)

type contextKey string

const (
	ClaimsContextKey contextKey = "claims"

	// Permissions
	CreateURLs    = "create:urls"
	DeleteURLs    = "delete:urls"
	DeleteOwnURLs = "delete:own-urls"
	GetOwnURLs    = "get:own-urls"
	GetURL        = "get:url"
)

// CustomClaims contains custom data we want from the token
type CustomClaims struct {
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
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

// HasPermission checks whether claims have a specific permission
func (c CustomClaims) HasPermission(expectedPermission string) bool {
	return slices.Contains(c.Permissions, expectedPermission)
}

func GetUserID(c echo.Context) *string {
	_, span := tracer.Start(c.Request().Context(), "auth.GetUserID")
	defer span.End()

	claims := getClaimsFromContext(c)
	if claims == nil || claims.RegisteredClaims.Subject == "" {
		span.SetAttributes(attribute.String("userID", ""))
		return nil
	}
	span.SetAttributes(attribute.String("userID", claims.RegisteredClaims.Subject))

	return &claims.RegisteredClaims.Subject
}

func setClaimsToContext(c echo.Context, claims any) {
	c.Set(string(ClaimsContextKey), claims)
}

func getClaimsFromContext(c echo.Context) *validator.ValidatedClaims {
	claims, ok := c.Get(string(ClaimsContextKey)).(*validator.ValidatedClaims)
	if !ok || claims == nil {
		return nil
	}

	return claims
}
