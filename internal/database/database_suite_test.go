package database

import (
	"context"
	"io"
	"testing"

	"github.com/rousage/shortener/internal/testhelpers"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

type DatabaseSuite struct {
	suite.Suite
	container *testhelpers.PostgresContainer
	ctx       context.Context
}

func (suite *DatabaseSuite) SetupSuite() {
	suite.ctx = context.Background()

	pgContainer, err := testhelpers.CreatePostgresContainer()
	if err != nil {
		log.Fatal().Msgf("could not start postgres container: %s", err)
	}
	suite.container = pgContainer
}

func (suite *DatabaseSuite) TearDownSuite() {
	if err := suite.container.Terminate(suite.ctx); err != nil {
		log.Fatal().Msgf("error terminating postgres container: %s", err)
	}
}

func (suite *DatabaseSuite) TestConnect() {
	logger := zerolog.New(io.Discard)
	db := Connect(logger, suite.container.DatabaseConfig)
	suite.NotNil(db, "Connect() returned nil")
}

func TestDatabaseSuite(t *testing.T) {
	suite.Run(t, new(DatabaseSuite))
}
