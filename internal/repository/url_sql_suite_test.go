package repository

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rousage/shortener/internal/database"
	"github.com/rousage/shortener/internal/testhelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UrlTestSuite struct {
	suite.Suite
	container *testhelpers.PostgresContainer
	db        *pgxpool.Pool
	queries   *Queries
	ctx       context.Context
}

func (suite *UrlTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Create a new postgres container for the whole test suite
	pgContainer, err := testhelpers.CreatePostgresContainer(suite.ctx)
	suite.Require().NoError(err, "could not start postgres container")

	// Snapshot the DB to restore it later
	err = pgContainer.Snapshot(suite.ctx)
	suite.Require().NoError(err)

	suite.container = pgContainer
}

func (suite *UrlTestSuite) TearDownSuite() {
	err := suite.container.Terminate(suite.ctx)
	suite.Require().NoError(err, "error terminating postgres container")
}

func (suite *UrlTestSuite) SetupTest() {
	// Connect to the DB before each test
	logger := zerolog.New(io.Discard)
	db := database.Connect(logger, suite.container.DatabaseConfig)
	queries := New(db)

	suite.db = db
	suite.queries = queries
}

func (suite *UrlTestSuite) TearDownTest() {
	// Restore the DB after each test to have a clean state
	suite.db.Close()
	err := suite.container.Restore(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *UrlTestSuite) TestCreateUrl() {
	t := suite.T()

	url, err := suite.queries.CreateUrl(suite.ctx, CreateUrlParams{ID: "short-url", LongUrl: "https://long.url"})
	assert.NoError(t, err)
	assert.Equal(t, "short-url", url.ID)
	assert.Equal(t, "https://long.url", url.LongUrl)
	assert.False(t, url.IsCustom)

	url2, err := suite.queries.CreateUrl(suite.ctx, CreateUrlParams{ID: "short-url2", LongUrl: "https://long.url", IsCustom: true})
	assert.NoError(t, err)
	assert.Equal(t, "short-url2", url2.ID)
	assert.Equal(t, "https://long.url", url2.LongUrl)
	assert.True(t, url2.IsCustom)

	_, err = suite.queries.CreateUrl(suite.ctx, CreateUrlParams{ID: "short-url", LongUrl: "https://another-long.url"})
	if assert.Error(t, err) {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			assert.Equal(t, "23505", pgErr.Code)
			assert.Equal(t, "duplicate key value violates unique constraint \"urls_pkey\"", pgErr.Message)
		}
	}

	_, err = suite.queries.CreateUrl(suite.ctx, CreateUrlParams{ID: "short-url-that-violaters-varchar-length", LongUrl: "https://another-long.url"})
	if assert.Error(t, err) {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			assert.Equal(t, "22001", pgErr.Code)
			assert.Equal(t, "value too long for type character varying(16)", pgErr.Message)
		}
	}
}

func (suite *UrlTestSuite) TestGetLongUrl() {
	t := suite.T()

	_, err := suite.queries.GetLongUrl(suite.ctx, "short-url")
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	_, err = suite.queries.CreateUrl(suite.ctx, CreateUrlParams{ID: "short-url", LongUrl: "https://long.url"})
	assert.NoError(t, err)

	longUrl, err := suite.queries.GetLongUrl(suite.ctx, "short-url")
	assert.NoError(t, err)
	assert.Equal(t, "https://long.url", longUrl)
}

func (suite *UrlTestSuite) TestDeleteShortUrl() {
	t := suite.T()

	rowsAffected, err := suite.queries.DeleteUrl(suite.ctx, "short-url")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), rowsAffected)

	_, err = suite.queries.CreateUrl(suite.ctx, CreateUrlParams{ID: "short-url", LongUrl: "https://long.url"})
	assert.NoError(t, err)

	rowsAffected, err = suite.queries.DeleteUrl(suite.ctx, "short-url")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)
}

func TestUrlTestSuite(t *testing.T) {
	suite.Run(t, new(UrlTestSuite))
}
