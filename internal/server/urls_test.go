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

	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/appvalidator"
	"github.com/rousage/shortener/internal/auth"
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
		expectedIsCustom       bool
	}{
		{name: "valid URL", payload: map[string]string{"url": "https://example.com"}, expectedStatus: http.StatusCreated, expectedUrl: "https://example.com", expectedShortUrlLength: 8, expectedIsCustom: false},
		{name: "valid URL with www", payload: map[string]string{"url": "https://www.example.com"}, expectedStatus: http.StatusCreated, expectedUrl: "https://www.example.com", expectedShortUrlLength: 8, expectedIsCustom: false},
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

			if tt.expectedStatus == http.StatusCreated {
				var actual repository.Url
				err = json.NewDecoder(res.Body).Decode(&actual)
				require.NoError(t, err, "error decoding response body")
				assert.Len(t, actual.ID, tt.expectedShortUrlLength, fmt.Sprintf("short URL should be %d characters long", tt.expectedShortUrlLength))
				assert.Equal(t, tt.expectedUrl, actual.LongUrl, "long URL does not match")
				assert.Equal(t, tt.expectedIsCustom, actual.IsCustom, "isCustom does not match")
			}
		})
	}

	t.Cleanup(cleanup)
}

func TestCreateShortURLHandler_IdenticalURLs(t *testing.T) {
	s, e, cleanup := setupTestServer(t)

	payload := CreateShortUrlDTO{URL: "https://example.com"}
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
	assert.Equal(t, payload.URL, actual1.LongUrl, "long URL does not match")
	assert.Equal(t, false, actual1.IsCustom, "isCustom does not match")

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
	assert.Equal(t, payload.URL, actual2.LongUrl, "long URL does not match")
	assert.Equal(t, false, actual2.IsCustom, "isCustom does not match")

	t.Cleanup(cleanup)
}

func TestCreateShortURLHandler_CustomShortCode(t *testing.T) {
	s, e, cleanup := setupTestServer(t)

	longUrl := "https://www.example.com"
	tests := []struct {
		name                string
		payload             CreateShortUrlDTO
		userId              string
		expectedStatus      int
		expectedUrl         string
		expectedShortUrlLen int
		expectedIsCustom    bool
	}{
		{name: "valid short code", payload: CreateShortUrlDTO{URL: longUrl, ShortCode: "short-Code_1"}, userId: "user-id", expectedStatus: http.StatusCreated, expectedUrl: longUrl, expectedShortUrlLen: 12, expectedIsCustom: true},
		{name: "duplicate short code", payload: CreateShortUrlDTO{URL: longUrl, ShortCode: "short-Code_1"}, userId: "user-id-1", expectedStatus: http.StatusConflict},
		{name: "too small short code", payload: CreateShortUrlDTO{URL: longUrl, ShortCode: "code"}, userId: "user-id", expectedStatus: http.StatusBadRequest},
		{name: "too big short code", payload: CreateShortUrlDTO{URL: longUrl, ShortCode: "short-code_1234567891"}, userId: "user-id", expectedStatus: http.StatusBadRequest},
		{name: "invalid short code", payload: CreateShortUrlDTO{URL: longUrl, ShortCode: "short-code$&*"}, userId: "user-id", expectedStatus: http.StatusBadRequest},
		// if no custom short code is provided, it will be generated, hence isCustom = false
		{name: "empty short code", payload: CreateShortUrlDTO{URL: longUrl}, userId: "user-id", expectedStatus: http.StatusCreated, expectedUrl: longUrl, expectedShortUrlLen: 8, expectedIsCustom: false},
		// if user is not authenticated, they cannot create custom short codes
		{name: "unauthenticated user", payload: CreateShortUrlDTO{URL: longUrl, ShortCode: "short-Code_1"}, expectedStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.payload)
			require.NoError(t, err, "could not marshal payload")

			req := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewBuffer(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)

			// Set userId claim (Subject) to the context
			if tt.userId != "" {
				c.Set(string(auth.ClaimsContextKey), &validator.ValidatedClaims{RegisteredClaims: validator.RegisteredClaims{
					Subject: tt.userId,
				}})
			}

			// Assertions
			err = s.CreateShortURLHandler(c)

			if tt.expectedStatus == http.StatusForbidden {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.Code)
			}

			if tt.expectedStatus == http.StatusCreated {
				var actual repository.Url
				err = json.NewDecoder(res.Body).Decode(&actual)
				require.NoError(t, err, "error decoding response body")
				assert.Len(t, actual.ID, tt.expectedShortUrlLen, "incorrect short URL length")
				assert.Equal(t, tt.expectedUrl, actual.LongUrl, "long URL does not match")
				assert.Equal(t, tt.expectedIsCustom, actual.IsCustom, "isCustom does not match")
				assert.Equal(t, tt.userId, *actual.UserID, "userID does not match")
			}
		})
	}

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
			actualCache, err := s.cache.GetLongUrl(c.Request().Context(), tt.code)
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

func TestDeleteShortUrlHandler(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	createdUrl := createShortUrl(t, s, e, "https://example.com")

	tests := []struct {
		name           string
		code           string
		expectedStatus int
		expectedCache  string
	}{
		{name: "non-existent code", code: "non-existent", expectedStatus: http.StatusNotFound},
		{name: "successful delete", code: createdUrl.ID, expectedStatus: http.StatusNoContent},
		{name: "nothing to delete", code: createdUrl.ID, expectedStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/urls/%s", tt.code), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)
			c.SetPath("/urls/:code")
			c.SetParamNames("code")
			c.SetParamValues(tt.code)

			// Assertions
			err := s.DeletShortUrlHandler(c)
			if err != nil {
				switch tt.expectedStatus {
				case http.StatusNotFound:
					assert.Equal(t, echo.ErrNotFound, err)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.Code)

				actualCache, err := s.cache.GetLongUrl(c.Request().Context(), tt.code)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCache, actualCache, "cache does not match")
			}

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
	cacheClient := cache.Connect(logger, cacheContainer.CacheConfig)
	require.NotNil(t, cacheClient, "could not connect to cache")

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
		cache:  cache.New(cacheClient),
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
	payload := CreateShortUrlDTO{URL: url}
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
