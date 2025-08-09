package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-glide/go/v2/options"
)

var (
	defaultExpire = 24 * time.Hour
)

func (c *Cache) SetLongUrl(ctx context.Context, code, longUrl string) (key string, err error) {
	key = c.getUrlKey(code)

	opts := options.NewSetOptions().SetExpiry(options.NewExpiryIn(defaultExpire))
	if _, err := c.client.SetWithOptions(ctx, key, longUrl, *opts); err != nil {
		return key, err
	}

	return key, nil
}

func (c *Cache) GetLongUrl(ctx context.Context, code string) (string, error) {
	key := c.getUrlKey(code)

	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return "", err
	}

	return resp.Value(), nil
}

func (c *Cache) getUrlKey(code string) string {
	return fmt.Sprintf("long_url:%s", code)
}
