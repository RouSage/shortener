package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/appvalidator"
	"github.com/rousage/shortener/internal/cache"
	"github.com/rousage/shortener/internal/config"
	"github.com/rousage/shortener/internal/database"
	"github.com/rousage/shortener/internal/repository"
	"github.com/rousage/shortener/internal/testhelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateShortURLHandler(t *testing.T) {
	s, e, cleanup := setupTestServer(t)

	tests := []struct {
		name                   string
		payload                map[string]string
		expectedStatus         int
		expectedUrl            string
		expectedShortUrlLength int
	}{
		{name: "valid URL", payload: map[string]string{"url": "https://example.com"}, expectedStatus: http.StatusCreated, expectedUrl: "https://example.com", expectedShortUrlLength: 8},
		{name: "valid URL with www", payload: map[string]string{"url": "https://www.example.com"}, expectedStatus: http.StatusCreated, expectedUrl: "https://www.example.com", expectedShortUrlLength: 8},
		{name: "invalid URL with www", payload: map[string]string{"url": "www.example.com"}, expectedStatus: http.StatusBadRequest},
		{name: "invalid URL", payload: map[string]string{"url": "test"}, expectedStatus: http.StatusBadRequest},
		{name: "invalid payload", payload: map[string]string{"notUrl": "test"}, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.payload)
			require.NoError(t, err, "could not marshal payload")

			req := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewBuffer(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)

			// Assertions
			err = s.CreateShortURLHandler(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, res.Code)

			var actual repository.Url
			err = json.NewDecoder(res.Body).Decode(&actual)
			require.NoError(t, err, "error decoding response body")
			assert.Len(t, actual.ID, tt.expectedShortUrlLength, fmt.Sprintf("short URL should be %d characters long", tt.expectedShortUrlLength))
			assert.Equal(t, tt.expectedUrl, actual.LongUrl, "long URL does not match")
		})
	}

	t.Cleanup(cleanup)
}

func TestCreateShortURLHandler_IdenticalURLs(t *testing.T) {
	s, e, cleanup := setupTestServer(t)

	payload := map[string]string{"url": "https://example.com"}
	body, err := json.Marshal(payload)
	require.NoError(t, err, "could not marshal payload")

	req1 := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewBuffer(body))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, resp1)

	// Assertions
	err = s.CreateShortURLHandler(c1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp1.Code)

	var actual1 repository.Url
	err = json.NewDecoder(resp1.Body).Decode(&actual1)
	require.NoError(t, err, "error decoding response body")
	assert.Len(t, actual1.ID, 8, "short URL ID should be 8 characters long")
	assert.Equal(t, payload["url"], actual1.LongUrl, "long URL does not match")

	// Make a second request with the same URL
	req2 := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewBuffer(body))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, resp2)

	// Assertions
	err = s.CreateShortURLHandler(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp2.Code)

	var actual2 repository.Url
	err = json.NewDecoder(resp2.Body).Decode(&actual2)
	require.NoError(t, err, "error decoding response body")
	assert.Len(t, actual2.ID, 8, "short URL ID should be 8 characters long")
	assert.NotEqual(t, actual1.ID, actual2.ID, "short URL IDs should be different")
	assert.Equal(t, payload["url"], actual2.LongUrl, "long URL does not match")

	t.Cleanup(cleanup)
}

func TestGetLongUrlHandler(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	createdUrl := createShortUrl(t, s, e, "https://example.com")

	tests := []struct {
		name           string
		code           string
		expectedStatus int
		expectedUrl    string
	}{
		{name: "valid code", code: createdUrl.ID, expectedStatus: http.StatusOK, expectedUrl: createdUrl.LongUrl},
		{name: "invalid code", code: "invalid", expectedStatus: http.StatusNotFound},
		{name: "empty code", code: "", expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/urls/%s", tt.code), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)
			c.SetPath("/urls/:code")
			c.SetParamNames("code")
			c.SetParamValues(tt.code)

			// Assertions
			err := s.GetLongUrlHandler(c)
			if err != nil {
				switch tt.expectedStatus {
				case http.StatusNotFound:
					assert.Equal(t, echo.ErrNotFound, err)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.Code)

				if tt.expectedStatus == http.StatusBadRequest {
					return
				}

				var actual map[string]string
				err = json.NewDecoder(res.Body).Decode(&actual)
				require.NoError(t, err, "error decoding response body")
				assert.Equal(t, tt.expectedUrl, actual["longUrl"], "long URL does not match")
			}
		})
	}

	t.Cleanup(cleanup)
}

func TestGetLongUrlHandler_Cache(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	createdUrl := createShortUrl(t, s, e, "https://example.com")

	tests := []struct {
		name           string
		code           string
		expectedStatus int
		expectedUrl    string
		expectedCache  string
	}{
		{name: "cache miss", code: createdUrl.ID, expectedStatus: http.StatusOK, expectedUrl: createdUrl.LongUrl, expectedCache: ""},
		{name: "cache hit", code: createdUrl.ID, expectedStatus: http.StatusOK, expectedUrl: createdUrl.LongUrl, expectedCache: createdUrl.LongUrl},
		{name: "cache hit 2", code: createdUrl.ID, expectedStatus: http.StatusOK, expectedUrl: createdUrl.LongUrl, expectedCache: createdUrl.LongUrl},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/urls/%s", tt.code), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)
			c.SetPath("/urls/:code")
			c.SetParamNames("code")
			c.SetParamValues(tt.code)

			// Assertions
			cache := cache.New(s.cache)
			actualCache, err := cache.GetLongUrl(c.Request().Context(), tt.code)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCache, actualCache, "cache does not match")

			err = s.GetLongUrlHandler(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, res.Code)

			var actual map[string]string
			err = json.NewDecoder(res.Body).Decode(&actual)
			require.NoError(t, err, "error decoding response body")
			assert.Equal(t, tt.expectedUrl, actual["longUrl"], "long URL does not match")

		})
	}

	t.Cleanup(cleanup)
}

func setupTestServer(t *testing.T) (*Server, *echo.Echo, func()) {
	ctx := context.Background()
	logger := zerolog.New(io.Discard)

	pgContainer, err := testhelpers.CreatePostgresContainer(ctx)
	require.NoError(t, err, "could not start postgres container")
	cacheContainer, err := testhelpers.CreateValkeyContainer(ctx)
	require.NoError(t, err, "could not start cache container")

	db := database.Connect(logger, pgContainer.DatabaseConfig)
	require.NotNil(t, db, "could not connect to postgres")
	cache := cache.Connect(logger, cacheContainer.CacheConfig)
	require.NotNil(t, cache, "could not connect to cache")

	cfg := &config.Config{
		Database: pgContainer.DatabaseConfig,
		Cache:    cacheContainer.CacheConfig,
		App: config.App{
			Env: config.EnvDevelopment,
		},
	}

	e := echo.New()
	e.Validator = appvalidator.New()

	s := &Server{
		logger: logger,
		cfg:    cfg,
		db:     db,
		cache:  cache,
	}

	cleanup := func() {
		err := pgContainer.Terminate(ctx)
		require.NoError(t, err, "error terminating postgres container")

		err = cacheContainer.Terminate(ctx)
		require.NoError(t, err, "error terminating cache container")
	}

	return s, e, cleanup
}

func createShortUrl(t *testing.T, s *Server, e *echo.Echo, url string) repository.Url {
	payload := map[string]string{"url": url}
	body, err := json.Marshal(payload)
	require.NoError(t, err, "could not marshal payload")

	createReq := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewBuffer(body))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRes := httptest.NewRecorder()
	createCtx := e.NewContext(createReq, createRes)

	err = s.CreateShortURLHandler(createCtx)
	require.NoError(t, err)

	var createdUrl repository.Url
	err = json.NewDecoder(createRes.Body).Decode(&createdUrl)
	require.NoError(t, err, "error decoding response body")

	return createdUrl
}
