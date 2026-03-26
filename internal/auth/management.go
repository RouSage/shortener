package auth

import (
	"context"
	"log/slog"
	"os"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/auth0/go-auth0/v2/management/client"
	"github.com/auth0/go-auth0/v2/management/option"
	"github.com/rousage/shortener/internal/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Management struct {
	client *client.Management
}

func NewManagement(logger *slog.Logger, cfg config.Auth) *Management {
	client, err := client.New(cfg.Auth0Domain, option.WithClientCredentials(context.Background(), cfg.Auth0ClientID, cfg.Auth0ClientSecret))
	if err != nil {
		logger.Error("failed to creacte Auth0 Management client", "error", err)
		os.Exit(1)
	}

	return &Management{
		client: client,
	}
}

func (m *Management) BlockUser(ctx context.Context, userID string) (*management.UpdateUserResponseContent, error) {
	ctx, span := tracer.Start(ctx, "auth.BlockUser")
	defer span.End()

	span.SetAttributes(attribute.String("userID", userID))

	updated, err := m.client.Users.Update(ctx, userID, &management.UpdateUserRequestContent{
		Blocked: management.Bool(true),
	})
	if err != nil {
		span.SetStatus(codes.Error, "failed to block the user")
		span.RecordError(err)
		return &management.UpdateUserResponseContent{}, err
	}

	return updated, nil
}

func (m *Management) UnblockUser(ctx context.Context, userID string) error {
	ctx, span := tracer.Start(ctx, "auth.UnblockUser")
	defer span.End()

	span.SetAttributes(attribute.String("userID", userID))

	_, err := m.client.Users.Update(ctx, userID, &management.UpdateUserRequestContent{
		Blocked: management.Bool(false),
	})
	if err != nil {
		span.SetStatus(codes.Error, "failed to unblock the user")
		span.RecordError(err)
		return err
	}

	// Also remove the user's brute-force protection block if any exists
	err = m.client.UserBlocks.Delete(ctx, userID)
	if err != nil {
		span.SetStatus(codes.Error, "failed to remove the user's brute-force protection block")
		span.RecordError(err)
		return err
	}

	return nil
}
