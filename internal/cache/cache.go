package cache

import (
	"context"
	"log/slog"
	"os"

	"github.com/rousage/shortener/internal/config"
	glide "github.com/valkey-io/valkey-glide/go/v2"
	cacheConfig "github.com/valkey-io/valkey-glide/go/v2/config"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/rousage/shortener/internal/cache")

func Connect(logger *slog.Logger, cfg config.Cache) *glide.Client {
	clientCfg := cacheConfig.NewClientConfiguration().WithAddress(&cacheConfig.NodeAddress{
		Host: cfg.Host,
		Port: cfg.Port,
	})

	client, err := glide.NewClient(clientCfg)
	if err != nil {
		logger.Error("faild to connect to cache", "error", err)
		os.Exit(1)
	}

	res, err := client.Ping(context.Background())
	if err != nil {
		logger.Error("failed to ping cache", "error", err)
		os.Exit(1)
	}
	logger.Debug("cache response", slog.String("response", res))

	return client
}

type Cache struct {
	client *glide.Client
}

func New(client *glide.Client) *Cache {
	return &Cache{client: client}
}
