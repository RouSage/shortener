package cache

import (
	"context"
	"io"
	"testing"

	"github.com/rousage/shortener/internal/testhelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CacheTestSuite struct {
	suite.Suite
	container *testhelpers.ValkeyContainer
	ctx       context.Context
}

func (suite *CacheTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	cacheContainer, err := testhelpers.CreateValkeyContainer(suite.ctx)
	require.NoError(suite.T(), err, "could not start cache container")

	suite.container = cacheContainer
}

func (suite *CacheTestSuite) TearDownSuite() {
	err := suite.container.Terminate(suite.ctx)
	require.NoError(suite.T(), err, "error terminating cache container")
}

func (suite *CacheTestSuite) TestConnect() {
	logger := zerolog.New(io.Discard)
	cache := Connect(logger, suite.container.CacheConfig)
	suite.NotNil(cache, "Connect() returned nil")
}

func TestCacheTestSuite(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}
