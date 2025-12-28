package auth

import (
	"context"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/auth0/go-auth0/v2/management/client"
	"github.com/auth0/go-auth0/v2/management/option"
	"github.com/rousage/shortener/internal/config"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Management struct {
	mgmt *client.Management
}

func NewManagement(cfg config.Auth, logger zerolog.Logger) *Management {
	mgmt, err := client.New(cfg.Auth0Domain, option.WithClientCredentials(context.Background(), cfg.Auth0ClientID, cfg.Auth0ClientSecret))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create Auth0 management client")
	}

	return &Management{
		mgmt: mgmt,
	}
}

func (m *Management) BlockUser(ctx context.Context, userID string) error {
	ctx, span := tracer.Start(ctx, "auth.BlockUser")
	defer span.End()

	span.SetAttributes(attribute.String("userID", userID))

	trueVal := true
	_, err := m.mgmt.Users.Update(ctx, userID, &management.UpdateUserRequestContent{
		Blocked: &trueVal,
	})
	if err != nil {
		span.SetStatus(codes.Error, "failed to block the user")
		span.RecordError(err)
		return err
	}

	return nil
}
