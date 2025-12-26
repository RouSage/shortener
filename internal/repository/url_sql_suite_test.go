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

	var (
		userID      = "user-id"
		emptyUserID = ""
	)

	// TODO: add test with a long ID (16+ chars)
	tests := []struct {
		params CreateUrlParams
	}{
		{params: CreateUrlParams{ID: "short-url", LongUrl: "https://long.url"}},
		{params: CreateUrlParams{ID: "short-url2", LongUrl: "https://long.url"}},
		{params: CreateUrlParams{ID: "short-url3", LongUrl: "https://long.url", IsCustom: true, UserID: &userID}},
		{params: CreateUrlParams{ID: "short-url4", LongUrl: "https://long.url", IsCustom: true, UserID: &emptyUserID}},
	}

	for _, tt := range tests {
		url, err := suite.queries.CreateUrl(suite.ctx, tt.params)
		assert.NoError(t, err)
		assert.Equal(t, tt.params.ID, url.ID)
		assert.Equal(t, tt.params.LongUrl, url.LongUrl)
		assert.Equal(t, tt.params.IsCustom, url.IsCustom)
		assert.Equal(t, tt.params.UserID, url.UserID)
	}

	_, err := suite.queries.CreateUrl(suite.ctx, CreateUrlParams{ID: "short-url", LongUrl: "https://another-long.url"})
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

	_, err = suite.queries.CreateUrl(suite.ctx, CreateUrlParams{ID: "short-url-custom", LongUrl: "https://long.url", IsCustom: true})
	if assert.Error(t, err) {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			assert.Equal(t, "23514", pgErr.Code)
			assert.Equal(t, "new row for relation \"urls\" violates check constraint \"custom_urls_require_user\"", pgErr.Message)
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

func (suite *UrlTestSuite) TestGetUserUrls() {
	t := suite.T()

	var (
		userID_1 = "user-id"
		userID_2 = "user-id-2"
	)

	urlParams := []CreateUrlParams{
		{ID: "short-url", LongUrl: "https://long.url", UserID: &userID_1},
		{ID: "short-url2", LongUrl: "https://long.url", UserID: &userID_1},
		{ID: "short-url3", LongUrl: "https://long.url", UserID: &userID_1},
		{ID: "short-url4", LongUrl: "https://long.url", UserID: &userID_1},
		{ID: "short-url5", LongUrl: "https://long.url", UserID: &userID_1},
		{ID: "short-url6", LongUrl: "https://long.url", UserID: &userID_2},
		{ID: "short-url7", LongUrl: "https://long.url", UserID: &userID_2},
	}

	urls, err := suite.queries.GetUserUrls(suite.ctx, GetUserUrlsParams{UserID: &userID_1, Limit: 25, Offset: 0})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(urls))

	urls, err = suite.queries.GetUserUrls(suite.ctx, GetUserUrlsParams{UserID: &userID_2, Limit: 25, Offset: 0})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(urls))

	for _, arg := range urlParams {
		_, err := suite.queries.CreateUrl(suite.ctx, arg)
		assert.NoError(t, err)
	}

	urls, err = suite.queries.GetUserUrls(suite.ctx, GetUserUrlsParams{UserID: &userID_1, Limit: 25, Offset: 0})
	assert.NoError(t, err)
	assert.Equal(t, 5, len(urls))

	urls, err = suite.queries.GetUserUrls(suite.ctx, GetUserUrlsParams{UserID: &userID_2, Limit: 25, Offset: 0})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(urls))
}

func (suite *UrlTestSuite) TestDeleteShortUrl() {
	t := suite.T()

	userId := "user-id"

	rowsAffected, err := suite.queries.DeleteUserURL(suite.ctx, DeleteUserURLParams{ID: "short-url", UserID: &userId})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), rowsAffected)

	_, err = suite.queries.CreateUrl(suite.ctx, CreateUrlParams{ID: "short-url", LongUrl: "https://long.url", UserID: &userId})
	assert.NoError(t, err)

	rowsAffected, err = suite.queries.DeleteUserURL(suite.ctx, DeleteUserURLParams{ID: "short-url", UserID: &userId})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)
}

func TestUrlTestSuite(t *testing.T) {
	suite.Run(t, new(UrlTestSuite))
}
