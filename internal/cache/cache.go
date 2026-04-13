package cache

import (
	"context"
	"log/slog"
	"os"

	"github.com/rousage/shortener/internal/config"
	glide "github.com/valkey-io/valkey-glide/go/v2"
	cacheConfig "github.com/valkey-io/valkey-glide/go/v2/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const name = "github.com/rousage/shortener/internal/cache"

var (
	tracer = otel.Tracer(name)
	meter  = otel.Meter(name)
)

func Connect(logger *slog.Logger, cfg config.Cache) *glide.Client {
	clientCfg := cacheConfig.NewClientConfiguration().WithAddress(&cacheConfig.NodeAddress{
		Host: cfg.Host,
		Port: cfg.Port,
	})

	client, err := glide.NewClient(clientCfg)
	if err != nil {
		logger.Error("failed to connect to cache", "error", err)
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
	client            *glide.Client
	resolutionCounter metric.Int64Counter
}

func New(logger *slog.Logger, client *glide.Client) *Cache {
	counter, err := meter.Int64Counter(
		"cache.resolutions",
		metric.WithDescription("Number of cache lookups for short URL resolution, labeled by result"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		// conunter creation failure is not fatal
		logger.Warn("failed to create cache resolution counter", "error", err)
	}

	return &Cache{client: client, resolutionCounter: counter}
}
