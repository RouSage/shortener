package auth

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/config"
	"github.com/rs/zerolog"
)

type AuthMiddleware struct {
	cfg    config.Auth
	logger zerolog.Logger
}

func NewAuthMiddleware(cfg config.Auth, logger zerolog.Logger) *AuthMiddleware {
	return &AuthMiddleware{cfg: cfg, logger: logger}
}

// Authenticate is a middleware that will check the validity of the JWT if it is present
func (m *AuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	issuerURL, err := url.Parse(fmt.Sprintf("https://%s/", m.cfg.Auth0Domain))
	if err != nil {
		m.logger.Fatal().Err(err).Msg("Failed to parse the issuer url")
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{m.cfg.Auth0Audience},
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		m.logger.Fatal().Err(err).Msg("Failed to set up the jwt validator")
	}

	return func(c echo.Context) error {
		token, err := jwtmiddleware.AuthHeaderTokenExtractor(c.Request())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		// If token is not present, just continue to the next handler.
		if token == "" {
			return next(c)
		}

		// Otherwise, validate the token.
		tokenInfo, err := jwtValidator.ValidateToken(c.Request().Context(), token)
		if err != nil {
			// TODO: add tracing
			return echo.ErrUnauthorized
		}

		c.Set(string(ClaimsContextKey), tokenInfo)

		return next(c)
	}
}

func (m *AuthMiddleware) RequireAuthentication(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID, ok := GetUserID(c)
		if !ok || userID == "" {
			return echo.ErrUnauthorized
		}

		return next(c)
	}
}
