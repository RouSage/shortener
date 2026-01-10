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
type permission string

const (
	ClaimsContextKey contextKey = "claims"

	// Permissions
	CreateURLs    permission = "create:urls"
	DeleteURLs    permission = "delete:urls"
	DeleteOwnURLs permission = "delete:own-urls"
	GetOwnURLs    permission = "get:own-urls"
	GetURL        permission = "get:url"
	GetURLs       permission = "get:urls"
	UserBlock     permission = "user:block"
	UserUnblock   permission = "user:unblock"
	GetUserBlocks permission = "get:user-blocks"
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
func (c CustomClaims) HasPermission(expectedPermission permission) bool {
	return slices.Contains(c.Permissions, string(expectedPermission))
}

func GetUserID(c echo.Context) *string {
	_, span := tracer.Start(c.Request().Context(), "auth.GetUserID")
	defer span.End()

	claims := getClaimsFromContext(c)
	if claims == nil || claims.RegisteredClaims.Subject == "" {
		span.SetAttributes(attribute.String("currUserID", ""))
		return nil
	}
	span.SetAttributes(attribute.String("currUserID", claims.RegisteredClaims.Subject))

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

func getCustomClaimsFromContext(c echo.Context) *CustomClaims {
	claims := getClaimsFromContext(c)
	if claims == nil {
		return nil
	}

	customClaims, ok := claims.CustomClaims.(*CustomClaims)
	if !ok {
		return nil
	}

	return customClaims
}
