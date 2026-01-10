package repository

import (
	"context"
	"io"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rousage/shortener/internal/database"
	"github.com/rousage/shortener/internal/testhelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var (
	userID_1  = "user-id"
	userID_2  = "user-id-2"
	adminID   = "adminID"
	urlParams = []CreateUrlParams{
		{ID: "short-url1", LongUrl: "https://long.url", UserID: &userID_1, IsCustom: true},
		{ID: "short-url2", LongUrl: "https://long.url", UserID: &userID_1},
		{ID: "short-url3", LongUrl: "https://long.url", UserID: &userID_1},
		{ID: "short-url4", LongUrl: "https://long.url", UserID: &userID_1},
		{ID: "short-url5", LongUrl: "https://long.url", UserID: &userID_1},
		{ID: "short-url6", LongUrl: "https://long.url", UserID: &userID_2},
		{ID: "short-url7", LongUrl: "https://long.url", UserID: &userID_2, IsCustom: true},
		{ID: "short-url8", LongUrl: "https://long.url"},
		{ID: "short-url9", LongUrl: "https://long.url"},
		{ID: "short-url10", LongUrl: "https://long.url"},
	}
	blockUserParams = []BlockUserParams{
		{UserID: userID_1, BlockedBy: adminID},
		{UserID: "user-2", BlockedBy: adminID},
		{UserID: "user-3", BlockedBy: adminID},
		{UserID: "user-4", BlockedBy: adminID},
		{UserID: "user-5", BlockedBy: adminID},
		{UserID: "user-6", BlockedBy: adminID},
		{UserID: "user-7", BlockedBy: adminID},
		{UserID: "user-8", BlockedBy: adminID},
		{UserID: "user-9", BlockedBy: adminID},
		{UserID: "user-10", BlockedBy: adminID},
	}
)

type AdminTestSuite struct {
	suite.Suite
	container *testhelpers.PostgresContainer
	db        *pgxpool.Pool
	queries   *Queries
	ctx       context.Context
}

func (suite *AdminTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Create a new postgres container for the whole test suite
	pgContainer, err := testhelpers.CreatePostgresContainer(suite.ctx)
	suite.Require().NoError(err, "could not start postgres container")

	logger := zerolog.New(io.Discard)
	db := database.Connect(logger, pgContainer.DatabaseConfig)
	queries := New(db)

	// Populate the DB with some data for tests
	for _, arg := range urlParams {
		_, err := queries.CreateUrl(suite.ctx, arg)
		suite.Require().NoError(err, "error creating seed urls")
	}
	for _, arg := range blockUserParams {
		_, err := queries.BlockUser(suite.ctx, arg)
		suite.Require().NoError(err, "error creating seed user blocks")
	}
	db.Close()

	// Snapshot the DB to restore it later
	err = pgContainer.Snapshot(suite.ctx)
	suite.Require().NoError(err)

	suite.container = pgContainer
}

func (suite *AdminTestSuite) TearDownSuite() {
	err := suite.container.Terminate(suite.ctx)
	suite.Require().NoError(err, "error terminating postgres container")
}

func (suite *AdminTestSuite) SetupTest() {
	// Connect to the DB before each test
	logger := zerolog.New(io.Discard)
	db := database.Connect(logger, suite.container.DatabaseConfig)
	queries := New(db)

	suite.db = db
	suite.queries = queries
}

func (suite *AdminTestSuite) TearDownTest() {
	// Restore the DB after each test to have a clean state
	suite.db.Close()
	err := suite.container.Restore(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *AdminTestSuite) TestGetURLs() {
	t := suite.T()

	var (
		trueVal  = true
		falseVal = false
	)

	tests := []struct {
		name         string
		params       GetURLsParams
		expectedUrls int
	}{
		{name: "get all urls", params: GetURLsParams{Offset: 0, Limit: 25}, expectedUrls: 10},
		{name: "get all custom urls", params: GetURLsParams{IsCustom: &trueVal, Offset: 0, Limit: 25}, expectedUrls: 2},
		{name: "get all generic urls", params: GetURLsParams{IsCustom: &falseVal, Offset: 0, Limit: 25}, expectedUrls: 8},
		{name: "get all user urls", params: GetURLsParams{UserID: &userID_1, Offset: 0, Limit: 25}, expectedUrls: 5},
		{name: "get all user custom urls", params: GetURLsParams{IsCustom: &trueVal, UserID: &userID_1, Offset: 0, Limit: 25}, expectedUrls: 1},
		{name: "get all user generic urls", params: GetURLsParams{IsCustom: &falseVal, UserID: &userID_1, Offset: 0, Limit: 25}, expectedUrls: 4},
		{name: "return urls for offset=0 and limit=5", params: GetURLsParams{Offset: 0, Limit: 5}, expectedUrls: 5},
		{name: "return urls for offset=5 and limit=5", params: GetURLsParams{Offset: 5, Limit: 5}, expectedUrls: 5},
		{name: "return urls for offset=10 and limit=5", params: GetURLsParams{Offset: 10, Limit: 5}, expectedUrls: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := suite.queries.GetURLs(suite.ctx, tt.params)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedUrls, len(urls))
		})
	}
}

func (suite *AdminTestSuite) TestDeleteURL() {
	t := suite.T()

	tests := []struct {
		name                 string
		id                   string
		userId               string
		expectedRowsAffected int64
	}{
		{name: "delete non-existing url", id: "short-url", userId: "user-id", expectedRowsAffected: int64(0)},
		{name: "delete existing url", id: "short-url1", expectedRowsAffected: int64(1)},
		{name: "delete the same url again", id: "short-url1", expectedRowsAffected: int64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rowsAffected, err := suite.queries.DeleteURL(suite.ctx, tt.id)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRowsAffected, rowsAffected)
		})
	}
}

func (suite *AdminTestSuite) TestDeleteAllUserURLs() {
	t := suite.T()

	tests := []struct {
		name                 string
		userId               string
		expectedRowsAffected int
	}{
		{name: "delete all user 1 urls", userId: userID_1, expectedRowsAffected: 5},
		{name: "delete all user 2 urls", userId: userID_2, expectedRowsAffected: 2},
		{name: "nothing to delete for user 1", userId: userID_1, expectedRowsAffected: 0},
		{name: "nothing to delete for user 2", userId: userID_2, expectedRowsAffected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deletedRows, err := suite.queries.DeleteAllUserURLs(suite.ctx, tt.userId)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRowsAffected, len(deletedRows))
		})
	}
}

func (suite *AdminTestSuite) TestBlockUser() {
	t := suite.T()

	var (
		reason = "Test reason"
		email  = "test@example.com"
	)

	tests := []struct {
		name   string
		params BlockUserParams
	}{
		{name: "blocks a user", params: BlockUserParams{UserID: userID_1, BlockedBy: adminID, Reason: &reason}},
		{name: "blocks another user", params: BlockUserParams{UserID: userID_2, BlockedBy: adminID, UserEmail: &email}},
		{name: "blocks already blocked user", params: BlockUserParams{UserID: userID_2, BlockedBy: adminID, UserEmail: &email}},
		{name: "email is overriden", params: BlockUserParams{UserID: userID_2, BlockedBy: adminID}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userBlock, err := suite.queries.BlockUser(suite.ctx, tt.params)
			assert.NoError(t, err)
			assert.Equal(t, tt.params.UserID, userBlock.UserID, "user id should be the same")
			assert.Equal(t, tt.params.BlockedBy, userBlock.BlockedBy, "blocked by should be the same")
			assert.Equal(t, tt.params.Reason, userBlock.Reason, "reason should be the same")
			assert.Equal(t, tt.params.UserEmail, userBlock.UserEmail, "user email should be the same")
			assert.Nil(t, userBlock.UnblockedBy, "unblocked by should be nil")
			assert.Nil(t, userBlock.UnblockedAt, "unblocked at should be nil")
		})
	}
}

func (suite *AdminTestSuite) TestUnblockUser() {
	t := suite.T()

	tests := []struct {
		name   string
		params UnblockUserParams
	}{
		{name: "unblocks a user", params: UnblockUserParams{UserID: userID_1, UnblockedBy: adminID}},
		{name: "unblocks already unblocked user", params: UnblockUserParams{UserID: userID_1, UnblockedBy: adminID}},
		{name: "unblocks another user that is not blocked and errors", params: UnblockUserParams{UserID: userID_2, UnblockedBy: adminID}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userBlock, err := suite.queries.UnblockUser(suite.ctx, tt.params)
			if err != nil {
				assert.ErrorIs(t, err, pgx.ErrNoRows)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.params.UserID, userBlock.UserID, "user id should be the same")
				assert.Equal(t, &tt.params.UnblockedBy, userBlock.UnblockedBy, "unblocked by should be the same")

				assert.Equal(t, blockUserParams[0].BlockedBy, userBlock.BlockedBy, "blocked by should be the same")
				assert.Equal(t, blockUserParams[0].Reason, userBlock.Reason, "reason should be the same")
				assert.Equal(t, blockUserParams[0].UserEmail, userBlock.UserEmail, "user email should be the same")

			}
		})
	}
}

func (suite *AdminTestSuite) TestGetUserBlocks() {
	t := suite.T()

	tests := []struct {
		name           string
		params         GetUserBlocksParams
		expectedBlocks int
	}{
		{name: "get all urls", params: GetUserBlocksParams{Offset: 0, Limit: 25}, expectedBlocks: 10},
		{name: "return urls for offset=0 and limit=5", params: GetUserBlocksParams{Offset: 0, Limit: 5}, expectedBlocks: 5},
		{name: "return urls for offset=5 and limit=5", params: GetUserBlocksParams{Offset: 5, Limit: 5}, expectedBlocks: 5},
		{name: "return urls for offset=10 and limit=5", params: GetUserBlocksParams{Offset: 10, Limit: 5}, expectedBlocks: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userBlocks, err := suite.queries.GetUserBlocks(suite.ctx, tt.params)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBlocks, len(userBlocks))
		})
	}
}

func TestAdminTestSuite(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}
