package cache

import (
	"context"
	"io"
	"testing"

	"github.com/rousage/shortener/internal/testhelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
)

type UrlTestSuite struct {
	suite.Suite
	container *testhelpers.ValkeyContainer
	cache     *Cache
	ctx       context.Context
}

func (suite *UrlTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Create a new cachecontainer for the whole test suite
	valkeyContainer, err := testhelpers.CreateValkeyContainer(suite.ctx)
	suite.Require().NoError(err, "could not start cache container")

	suite.container = valkeyContainer
}

func (suite *UrlTestSuite) TearDownSuite() {
	suite.cache.client.Close()
	err := suite.container.Terminate(suite.ctx)
	suite.Require().NoError(err, "error terminating cache container")
}

func (suite *UrlTestSuite) SetupTest() {
	// Connect to the cache before each test
	logger := zerolog.New(io.Discard)
	client := Connect(logger, suite.container.CacheConfig)
	cache := New(client)

	suite.cache = cache
}

func (suite *UrlTestSuite) TearDownTest() {
	// Clean the cache after each test
	resp, err := suite.cache.client.FlushAll(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal("OK", resp)
}

func (suite *UrlTestSuite) TestSetLongUrl() {
	expectedTTL := int64(defaultExpire.Seconds())

	// Write a long URL to the cache and get bakc a key
	key, err := suite.cache.SetLongUrl(suite.ctx, "short-url", "https://long.url")
	suite.NoError(err)
	suite.Equal("long_url:short-url", key)
	// Check that TTL is set to default
	ttl, err := suite.cache.client.TTL(suite.ctx, key)
	suite.NoError(err)
	suite.GreaterOrEqual(ttl, expectedTTL-1, "incorrect TTL (too low)")
	suite.LessOrEqual(ttl, expectedTTL, "incorrect TTL (too high)")

	// Write another URL to the same key
	key, err = suite.cache.SetLongUrl(suite.ctx, "short-url", "https://new-long.url")
	suite.NoError(err)
	suite.Equal("long_url:short-url", key)
	// Make sure the TTL is still the default
	ttl, err = suite.cache.client.TTL(suite.ctx, key)
	suite.NoError(err)
	suite.GreaterOrEqual(ttl, expectedTTL-1, "incorrect TTL (too low)")
	suite.LessOrEqual(ttl, expectedTTL, "incorrect TTL (too high)")

	key2, err := suite.cache.SetLongUrl(suite.ctx, "short-url2", "https://another-long.url")
	suite.NoError(err)
	suite.Equal("long_url:short-url2", key2)

	ttl2, err := suite.cache.client.TTL(suite.ctx, key2)
	suite.NoError(err)
	suite.GreaterOrEqual(ttl2, expectedTTL-1, "incorrect TTL (too low)")
	suite.LessOrEqual(ttl2, expectedTTL, "incorrect TTL (too high)")

	resp, err := suite.cache.client.Exists(suite.ctx, []string{key, key2})
	suite.NoError(err)
	suite.Equal(int64(2), resp, "incorrect number of keys in cache")
}

func (suite *UrlTestSuite) TestGetLongUrl() {
	code := "short-url"

	longUrl, err := suite.cache.GetLongUrl(suite.ctx, code)
	suite.NoError(err)
	suite.Empty(longUrl, "long URL is not empty for non-existing cache entry")

	_, err = suite.cache.SetLongUrl(suite.ctx, code, "https://long.url")
	suite.NoError(err)

	longUrl, err = suite.cache.GetLongUrl(suite.ctx, code)
	suite.NoError(err)
	suite.Equal("https://long.url", longUrl, "long URL is not correct for existing cache entry")

	// Make sure the cache entry is overridden to a new value
	_, err = suite.cache.SetLongUrl(suite.ctx, code, "https://another-long.url")
	suite.NoError(err)

	longUrl, err = suite.cache.GetLongUrl(suite.ctx, code)
	suite.NoError(err)
	suite.Equal("https://another-long.url", longUrl, "long URL is not correct for existing cache entry")
}

func TestUrlTestSuite(t *testing.T) {
	suite.Run(t, new(UrlTestSuite))
}
