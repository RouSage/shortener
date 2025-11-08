package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-glide/go/v2/options"
	"go.opentelemetry.io/otel/attribute"
)

var (
	defaultExpire = 24 * time.Hour
)

func (c *Cache) SetLongUrl(ctx context.Context, code, longUrl string) (key string, err error) {
	ctx, span := tracer.Start(ctx, "cache.SetLongUrl")
	defer span.End()

	key = c.getUrlKey(code)
	span.SetAttributes(attribute.String("key", key))

	opts := options.NewSetOptions().SetExpiry(options.NewExpiryIn(defaultExpire))
	if _, err := c.client.SetWithOptions(ctx, key, longUrl, *opts); err != nil {
		span.RecordError(err)
		return key, err
	}

	return key, nil
}

func (c *Cache) GetLongUrl(ctx context.Context, code string) (string, error) {
	ctx, span := tracer.Start(ctx, "cache.GetLongUrl")
	defer span.End()

	key := c.getUrlKey(code)
	span.SetAttributes(attribute.String("key", key))

	resp, err := c.client.Get(ctx, key)
	if err != nil {
		span.RecordError(err)
		return "", err
	}

	if resp.Value() == "" {
		span.AddEvent("long url not found in cache")
	}

	return resp.Value(), nil
}

func (c *Cache) getUrlKey(code string) string {
	return fmt.Sprintf("long_url:%s", code)
}
