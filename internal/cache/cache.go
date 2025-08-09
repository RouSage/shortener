package cache

import (
	"context"

	"github.com/rousage/shortener/internal/config"
	"github.com/rs/zerolog"
	glide "github.com/valkey-io/valkey-glide/go/v2"
	cacheConfig "github.com/valkey-io/valkey-glide/go/v2/config"
)

func Connect(logger zerolog.Logger, cfg config.Cache) *glide.Client {
	clientCfg := cacheConfig.NewClientConfiguration().WithAddress(&cacheConfig.NodeAddress{
		Host: cfg.Host,
		Port: cfg.Port,
	})

	client, err := glide.NewClient(clientCfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to cache")
	}

	res, err := client.Ping(context.Background())
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to ping cache")
	}
	logger.Debug().Msgf("cache response: %s", res)

	return client
}

type Cache struct {
	client *glide.Client
}

func New(client *glide.Client) *Cache {
	return &Cache{client: client}
}
