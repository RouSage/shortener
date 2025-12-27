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

			req := httptest.NewRequest(http.MethodPost, "/v1/urls", bytes.NewBuffer(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)

			// Assertions
			err = s.createShortURLHandler(c)
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

	req1 := httptest.NewRequest(http.MethodPost, "/v1/urls", bytes.NewBuffer(body))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, resp1)

	// Assertions
	err = s.createShortURLHandler(c1)
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
	err = s.createShortURLHandler(c2)
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

			req := httptest.NewRequest(http.MethodPost, "/v1/urls", bytes.NewBuffer(body))
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
			err = s.createShortURLHandler(c)
			if he, ok := err.(*echo.HTTPError); ok {
				assert.Equal(t, tt.expectedStatus, he.Code)
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
	createdUrl := createShortUrl(t, s, e, "https://example.com", "", "")

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
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/urls/%s", tt.code), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)
			c.SetPath("/v1/urls/:code")
			c.SetParamNames("code")
			c.SetParamValues(tt.code)

			// Assertions
			err := s.getLongUrlHandler(c)
			if he, ok := err.(*echo.HTTPError); ok {
				assert.Equal(t, tt.expectedStatus, he.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.Code)

				if tt.expectedStatus == http.StatusBadRequest {
					return
				}

				var actual GetLongUrlResponse
				err = json.NewDecoder(res.Body).Decode(&actual)
				require.NoError(t, err, "error decoding response body")
				assert.Equal(t, tt.expectedUrl, actual.LongUrl, "long URL does not match")
			}
		})
	}

	t.Cleanup(cleanup)
}

func TestGetLongUrlHandler_Cache(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	createdUrl := createShortUrl(t, s, e, "https://example.com", "", "")

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
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/urls/%s", tt.code), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)
			c.SetPath("/v1/urls/:code")
			c.SetParamNames("code")
			c.SetParamValues(tt.code)

			// Assertions
			actualCache, err := s.cache.GetLongUrl(c.Request().Context(), tt.code)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCache, actualCache, "cache does not match")

			err = s.getLongUrlHandler(c)
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

func TestGetUserUrlsHandler(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	authMw := auth.NewAuthMiddleware(s.cfg.Auth, s.logger)

	var (
		userID_1 = "user-id"
		userID_2 = "user-id-2"
	)

	for i := range 5 {
		createShortUrl(t, s, e, fmt.Sprintf("https://example-%d.com", i), userID_1, "")
	}

	tests := []struct {
		name              string
		userID            string
		withoutPermission bool
		page              int
		pageSize          int
		expectedStatus    int
		expectedUrls      int
	}{
		{name: "return user urls", userID: userID_1, page: 1, pageSize: 25, expectedStatus: http.StatusOK, expectedUrls: 5},
		{name: "return user urls for page=1 and pageSize=3", userID: userID_1, page: 1, pageSize: 3, expectedStatus: http.StatusOK, expectedUrls: 3},
		{name: "return user urls for page=2 and pageSize=3", userID: userID_1, page: 2, pageSize: 3, expectedStatus: http.StatusOK, expectedUrls: 2},
		{name: "return user urls for page=3 and pageSize=3", userID: userID_1, page: 3, pageSize: 3, expectedStatus: http.StatusOK},
		{name: "return no urls for user without urls", page: 1, pageSize: 25, userID: userID_2, expectedStatus: http.StatusOK},
		{name: "error on 0 page", userID: userID_1, page: 0, pageSize: 25, expectedStatus: http.StatusBadRequest},
		{name: "error on page > max", userID: userID_1, page: 20000, pageSize: 25, expectedStatus: http.StatusBadRequest},
		{name: "error on 0 pageSize", userID: userID_1, page: 1, pageSize: 0, expectedStatus: http.StatusBadRequest},
		{name: "error on pageSize > max", userID: userID_1, page: 1, pageSize: 101, expectedStatus: http.StatusBadRequest},
		{name: "unauthenticated user", page: 1, pageSize: 101, expectedStatus: http.StatusUnauthorized},
		{name: "no required permission", page: 1, pageSize: 101, userID: userID_1, withoutPermission: true, expectedStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/urls?page=%d&pageSize=%d", tt.page, tt.pageSize), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)
			c.SetPath("/v1/urls")

			if tt.userID != "" {
				claims := &validator.ValidatedClaims{
					RegisteredClaims: validator.RegisteredClaims{Subject: tt.userID},
					CustomClaims:     &auth.CustomClaims{},
				}
				if !tt.withoutPermission {
					claims.CustomClaims.(*auth.CustomClaims).Permissions = []string{string(auth.GetOwnURLs)}
				}

				c.Set(string(auth.ClaimsContextKey), claims)
			}

			handler := authMw.RequireAuthentication(authMw.RequirePermission(auth.GetOwnURLs)(s.getUserUrls))

			// Assertions
			err := handler(c)
			if he, ok := err.(*echo.HTTPError); ok {
				assert.Equal(t, tt.expectedStatus, he.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.Code)

				var actual PaginatedUserURLs
				err = json.NewDecoder(res.Body).Decode(&actual)
				require.NoError(t, err, "error decoding response body")
				assert.Equal(t, tt.expectedUrls, len(actual.Items), "incorrect number of urls")
			}

		})
	}

	t.Cleanup(cleanup)
}

func TestDeleteShortUrlHandler(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	authMw := auth.NewAuthMiddleware(s.cfg.Auth, s.logger)

	userID := "user-id"
	createdUrl := createShortUrl(t, s, e, "https://example.com", userID, "")
	_, err := s.cache.SetLongUrl(context.Background(), createdUrl.ID, createdUrl.LongUrl)
	require.NoError(t, err)

	tests := []struct {
		name              string
		code              string
		userID            string
		withoutPermission bool
		expectedStatus    int
	}{
		{name: "non-existent code", code: "non-existent", userID: userID, expectedStatus: http.StatusNotFound},
		{name: "unauthenticated user", code: createdUrl.ID, expectedStatus: http.StatusUnauthorized},
		{name: "no required permission", code: createdUrl.ID, userID: userID, withoutPermission: true, expectedStatus: http.StatusForbidden},
		{name: "successful delete", code: createdUrl.ID, userID: userID, expectedStatus: http.StatusNoContent},
		{name: "nothing to delete", code: createdUrl.ID, userID: userID, expectedStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/urls/%s", tt.code), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()

			c := e.NewContext(req, res)
			c.SetPath("/v1/urls/:code")
			c.SetParamNames("code")
			c.SetParamValues(tt.code)

			if tt.userID != "" {
				claims := &validator.ValidatedClaims{
					RegisteredClaims: validator.RegisteredClaims{Subject: userID},
					CustomClaims:     &auth.CustomClaims{},
				}
				if !tt.withoutPermission {
					claims.CustomClaims.(*auth.CustomClaims).Permissions = []string{string(auth.DeleteOwnURLs)}
				}

				c.Set(string(auth.ClaimsContextKey), claims)
			}

			handler := authMw.RequireAuthentication(authMw.RequirePermission(auth.DeleteOwnURLs)(s.deletShortUrlHandler))

			// Assertions
			err := handler(c)
			if he, ok := err.(*echo.HTTPError); ok {
				assert.Equal(t, tt.expectedStatus, he.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.Code)

				actualCache, err := s.cache.GetLongUrl(c.Request().Context(), tt.code)
				require.NoError(t, err)
				assert.Equal(t, "", actualCache, "cache does not match")
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

func createShortUrl(t *testing.T, s *Server, e *echo.Echo, url string, userID string, shortCode string) repository.Url {
	payload := CreateShortUrlDTO{URL: url, ShortCode: shortCode}
	body, err := json.Marshal(payload)
	require.NoError(t, err, "could not marshal payload")

	createReq := httptest.NewRequest(http.MethodPost, "/v1/urls", bytes.NewBuffer(body))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRes := httptest.NewRecorder()
	createCtx := e.NewContext(createReq, createRes)

	if userID != "" {
		createCtx.Set(string(auth.ClaimsContextKey), &validator.ValidatedClaims{RegisteredClaims: validator.RegisteredClaims{
			Subject: userID,
		}})
	}

	err = s.createShortURLHandler(createCtx)
	require.NoError(t, err)

	var createdUrl repository.Url
	err = json.NewDecoder(createRes.Body).Decode(&createdUrl)
	require.NoError(t, err, "error decoding response body")

	return createdUrl
}
