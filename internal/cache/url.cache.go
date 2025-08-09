package cache

import (
	"context"
	"fmt"
	"time"
)

var (
	defaultExpire = 24 * time.Hour
)

func (c *Cache) SetLongUrl(ctx context.Context, code, longUrl string) (key string, err error) {
	key = c.getUrlKey(code)

	if _, err := c.client.Set(ctx, key, longUrl); err != nil {
		return key, err
	}
	if _, err := c.client.Expire(ctx, key, defaultExpire); err != nil {
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
