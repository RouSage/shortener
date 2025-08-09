package database

import (
	"context"
	"io"
	"testing"

	"github.com/rousage/shortener/internal/testhelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
)

type DatabaseTestSuite struct {
	suite.Suite
	container *testhelpers.PostgresContainer
	ctx       context.Context
}

func (suite *DatabaseTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	pgContainer, err := testhelpers.CreatePostgresContainer(suite.ctx)
	suite.Require().NoError(err, "could not start postgres container")

	suite.container = pgContainer
}

func (suite *DatabaseTestSuite) TearDownSuite() {
	err := suite.container.Terminate(suite.ctx)
	suite.Require().NoError(err, "error terminating postgres container")
}

func (suite *DatabaseTestSuite) TestConnect() {
	logger := zerolog.New(io.Discard)
	db := Connect(logger, suite.container.DatabaseConfig)
	suite.NotNil(db, "Connect() returned nil")
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
