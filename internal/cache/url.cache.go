package cache

import (
	"context"
	"fmt"
	"time"
)

var (
	defaultExpire = 24 * time.Hour
)

func (c *Cache) SetLongUrl(ctx context.Context, code, longUrl string) error {
	key := c.getUrlKey(code)

	if _, err := c.client.Set(ctx, key, longUrl); err != nil {
		return err
	}
	if _, err := c.client.Expire(ctx, key, defaultExpire); err != nil {
		return err
	}

	return nil
}

func (c *Cache) GetLongUrl(ctx context.Context, code string) (string, error) {
	key := c.getUrlKey(code)

	res, err := c.client.Get(ctx, key)
	if err != nil {
		return "", err
	}

	return res.Value(), nil
}

func (c *Cache) getUrlKey(code string) string {
	return fmt.Sprintf("long_url:%s", code)
}
