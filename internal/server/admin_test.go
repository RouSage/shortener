package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/auth0/go-auth0/v2/management/core"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/auth"
	"github.com/rousage/shortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	adminID  = "admin-id"
	userID_1 = "user-id"
	userID_2 = "user-id-2"
	trueVal  = true
	falseVal = false
)

// mockAuthManager is a test implementation of AuthManager interface using testify/mock
type mockAuthManager struct {
	mock.Mock
}

func (m *mockAuthManager) BlockUser(ctx context.Context, userID string) (*management.UpdateUserResponseContent, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*management.UpdateUserResponseContent), args.Error(1)
}

func (m *mockAuthManager) UnblockUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestGetURLsHandler(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	authMw := auth.NewMiddleware(s.cfg.Auth, s.logger)

	for i := range 5 {
		createShortUrl(t, s, e, fmt.Sprintf("https://example-%d.com", i), "", "")
		createShortUrl(t, s, e, fmt.Sprintf("https://example-one-%d.com", i), userID_1, "")
		createShortUrl(t, s, e, fmt.Sprintf("https://example-two-%d.com", i), userID_2, fmt.Sprintf("custom-code-%d", i))
	}

	tests := []struct {
		name              string
		withoutPermission bool
		filters           URLsFilters
		expectedStatus    int
		expectedUrls      int
	}{
		{name: "no required permission", filters: URLsFilters{PaginationFilters: PaginationFilters{Page: 1, PageSize: 101}}, withoutPermission: true, expectedStatus: http.StatusForbidden},
		{name: "return all urls", filters: URLsFilters{PaginationFilters: PaginationFilters{Page: 1, PageSize: 25}}, expectedStatus: http.StatusOK, expectedUrls: 15},
		{name: "return all custom urls", filters: URLsFilters{IsCustom: &trueVal, PaginationFilters: PaginationFilters{Page: 1, PageSize: 25}}, expectedStatus: http.StatusOK, expectedUrls: 5},
		{name: "return all generic urls", filters: URLsFilters{IsCustom: &falseVal, PaginationFilters: PaginationFilters{Page: 1, PageSize: 25}}, expectedStatus: http.StatusOK, expectedUrls: 10},
		{name: "return all user urls", filters: URLsFilters{UserID: &userID_1, PaginationFilters: PaginationFilters{Page: 1, PageSize: 25}}, expectedStatus: http.StatusOK, expectedUrls: 5},
		{name: "return all user custom urls (no custom)", filters: URLsFilters{IsCustom: &trueVal, UserID: &userID_1, PaginationFilters: PaginationFilters{Page: 1, PageSize: 25}}, expectedStatus: http.StatusOK, expectedUrls: 0},
		{name: "return all user custom urls (has custom)", filters: URLsFilters{IsCustom: &trueVal, UserID: &userID_2, PaginationFilters: PaginationFilters{Page: 1, PageSize: 25}}, expectedStatus: http.StatusOK, expectedUrls: 5},
		{name: "return urls for page=1 and pageSize=5", filters: URLsFilters{PaginationFilters: PaginationFilters{Page: 1, PageSize: 5}}, expectedStatus: http.StatusOK, expectedUrls: 5},
		{name: "return urls for page=3 and pageSize=5", filters: URLsFilters{PaginationFilters: PaginationFilters{Page: 3, PageSize: 5}}, expectedStatus: http.StatusOK, expectedUrls: 5},
		{name: "return urls for page=4 and pageSize=5", filters: URLsFilters{PaginationFilters: PaginationFilters{Page: 4, PageSize: 5}}, expectedStatus: http.StatusOK},
		{name: "error on 0 page", filters: URLsFilters{PaginationFilters: PaginationFilters{Page: 0, PageSize: 25}}, expectedStatus: http.StatusBadRequest},
		{name: "error on page > max", filters: URLsFilters{PaginationFilters: PaginationFilters{Page: 20_000, PageSize: 25}}, expectedStatus: http.StatusBadRequest},
		{name: "error on 0 pageSize", filters: URLsFilters{PaginationFilters: PaginationFilters{Page: 1, PageSize: 0}}, expectedStatus: http.StatusBadRequest},
		{name: "error on pageSize > max", filters: URLsFilters{PaginationFilters: PaginationFilters{Page: 1, PageSize: 101}}, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/v1/admin/urls?page=%d&pageSize=%d", tt.filters.Page, tt.filters.PageSize)
			if tt.filters.IsCustom != nil {
				url += fmt.Sprintf("&isCustom=%t", *tt.filters.IsCustom)
			}
			if tt.filters.UserID != nil {
				url += fmt.Sprintf("&userId=%s", *tt.filters.UserID)
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)
			c.SetPath("/v1/admin/urls")

			claims := &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{Subject: adminID},
				CustomClaims:     &auth.CustomClaims{},
			}
			if !tt.withoutPermission {
				claims.CustomClaims.(*auth.CustomClaims).Permissions = []string{string(auth.GetURLs)}
			}
			c.Set(string(auth.ClaimsContextKey), claims)

			handler := authMw.RequireAuthentication(authMw.RequirePermission(auth.GetURLs)(s.getURLs))

			// Assertions
			err := handler(c)
			if he, ok := err.(*echo.HTTPError); ok {
				assert.Equal(t, tt.expectedStatus, he.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.Code)

				var actual PaginatedURLs
				err = json.NewDecoder(res.Body).Decode(&actual)
				require.NoError(t, err, "error decoding response body")
				assert.Equal(t, tt.expectedUrls, len(actual.Items), "incorrect number of urls")
			}
		})
	}

	t.Cleanup(cleanup)
}

func TestDeleteURLHandler(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	authMw := auth.NewMiddleware(s.cfg.Auth, s.logger)

	createdUrl := createShortUrl(t, s, e, "https://example.com", "", "")
	_, err := s.cache.SetLongUrl(context.Background(), createdUrl.ID, createdUrl.LongUrl)
	require.NoError(t, err)

	tests := []struct {
		name              string
		code              string
		withoutPermission bool
		expectedStatus    int
	}{
		{name: "no required permission", code: createdUrl.ID, withoutPermission: true, expectedStatus: http.StatusForbidden},
		{name: "non-existent code", code: "non-existent", expectedStatus: http.StatusNotFound},
		{name: "successful delete", code: createdUrl.ID, expectedStatus: http.StatusNoContent},
		{name: "nothing to delete", code: createdUrl.ID, expectedStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/admin/urls/%s", tt.code), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()

			c := e.NewContext(req, res)
			c.SetPath("/v1/admin/urls/:code")
			c.SetParamNames("code")
			c.SetParamValues(tt.code)

			claims := &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{Subject: adminID},
				CustomClaims:     &auth.CustomClaims{},
			}
			if !tt.withoutPermission {
				claims.CustomClaims.(*auth.CustomClaims).Permissions = []string{string(auth.DeleteURLs)}
			}
			c.Set(string(auth.ClaimsContextKey), claims)

			handler := authMw.RequireAuthentication(authMw.RequirePermission(auth.DeleteURLs)(s.deleteURLHandler))

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

func TestDeleteUserURLsHandler(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	authMw := auth.NewMiddleware(s.cfg.Auth, s.logger)

	invalidUserID := "the-user-id-that-is-too-long-for-the-endpoint-that-validation-should-prevent"
	// Populate DB and cache with some URLs
	codes_1 := make([]string, 5)
	codes_2 := make([]string, 5)
	for i := range 5 {
		url_1 := createShortUrl(t, s, e, fmt.Sprintf("https://example-one-%d.com", i), userID_1, "")
		url_2 := createShortUrl(t, s, e, fmt.Sprintf("https://example-two-%d.com", i), userID_2, fmt.Sprintf("custom-code-%d", i))
		_, err := s.cache.SetLongUrl(context.Background(), url_1.ID, url_1.LongUrl)
		require.NoError(t, err)
		_, err = s.cache.SetLongUrl(context.Background(), url_2.ID, url_2.LongUrl)
		require.NoError(t, err)

		codes_1[i] = url_1.ID
		codes_2[i] = url_2.ID
	}

	tests := []struct {
		name              string
		userId            string
		codes             []string
		withoutPermission bool
		expectedStatus    int
	}{
		{name: "no required permission", userId: userID_1, codes: codes_1, withoutPermission: true, expectedStatus: http.StatusForbidden},
		{name: "delete all user 1 urls", userId: userID_1, codes: codes_1, expectedStatus: http.StatusOK},
		{name: "delete all user 2 urls", userId: userID_2, codes: codes_2, expectedStatus: http.StatusOK},
		{name: "nothing to delete for user 1", userId: userID_1, codes: []string{}, expectedStatus: http.StatusOK},
		{name: "nothing to delete for user 2", userId: userID_2, codes: []string{}, expectedStatus: http.StatusOK},
		{name: "invalid user id", userId: invalidUserID, codes: []string{}, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/admin/urls/user/%s", tt.userId), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()

			c := e.NewContext(req, res)
			c.SetPath("/v1/admin/urls/user/:userId")
			c.SetParamNames("userId")
			c.SetParamValues(tt.userId)

			claims := &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{Subject: adminID},
				CustomClaims:     &auth.CustomClaims{},
			}
			if !tt.withoutPermission {
				claims.CustomClaims.(*auth.CustomClaims).Permissions = []string{string(auth.DeleteURLs)}
			}
			c.Set(string(auth.ClaimsContextKey), claims)

			handler := authMw.RequireAuthentication(authMw.RequirePermission(auth.DeleteURLs)(s.deleteUserURLsHandler))

			// Assertions
			err := handler(c)
			if he, ok := err.(*echo.HTTPError); ok {
				assert.Equal(t, tt.expectedStatus, he.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.Code)

				var actual DeleteUserURLsResponse
				err = json.NewDecoder(res.Body).Decode(&actual)
				require.NoError(t, err, "error decoding response body")
				assert.Equal(t, len(tt.codes), actual.Deleted, "incorrect number of deleted URLs")

				for _, code := range tt.codes {
					actualCache, err := s.cache.GetLongUrl(c.Request().Context(), code)
					require.NoError(t, err)
					assert.Equal(t, "", actualCache, "cache does not match")
				}
			}
		})
	}

	t.Cleanup(cleanup)
}

func TestBlockUserHandler(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	authMw := auth.NewMiddleware(s.cfg.Auth, s.logger)

	var (
		validUserID   = "auth0|507f1f77bcf86cd799439011"
		userEmail     = "user@example.com"
		invalidUserID = "user-id-that-is-way-too-long-and-exceeds-the-maximum-length-of-fifty-characters"
		reason        = "Test reason"
	)

	tests := []struct {
		name              string
		userID            string
		userEmail         string
		payload           BlockUserDTO
		withoutPermission bool
		mockSetup         func() *mockAuthManager
		expectedStatus    int
	}{
		{
			name:              "no required permission",
			userID:            validUserID,
			withoutPermission: true,
			mockSetup:         func() *mockAuthManager { return &mockAuthManager{} },
			expectedStatus:    http.StatusForbidden,
		},
		{
			name:   "successful block",
			userID: validUserID,
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("BlockUser", mock.Anything, validUserID).Return(&management.UpdateUserResponseContent{}, nil)
				return m
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:      "successful block (with email)",
			userID:    validUserID,
			userEmail: userEmail,
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("BlockUser", mock.Anything, validUserID).Return(&management.UpdateUserResponseContent{Email: &userEmail}, nil)
				return m
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "successful block (with reason)",
			userID:  validUserID,
			payload: BlockUserDTO{Reason: &reason},
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("BlockUser", mock.Anything, validUserID).Return(&management.UpdateUserResponseContent{}, nil)
				return m
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "user not found in Auth0 (404)",
			userID: validUserID,
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("BlockUser", mock.Anything, validUserID).Return(&management.UpdateUserResponseContent{}, &core.APIError{
					StatusCode: http.StatusNotFound,
				})
				return m
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "Auth0 returns 400 bad request",
			userID: validUserID,
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("BlockUser", mock.Anything, validUserID).Return(&management.UpdateUserResponseContent{}, &core.APIError{
					StatusCode: http.StatusBadRequest,
				})
				return m
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Auth0 returns 429 rate limit",
			userID: validUserID,
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("BlockUser", mock.Anything, validUserID).Return(&management.UpdateUserResponseContent{}, &core.APIError{
					StatusCode: http.StatusTooManyRequests,
				})
				return m
			},
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:   "non-API error returns 500",
			userID: validUserID,
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("BlockUser", mock.Anything, validUserID).Return(&management.UpdateUserResponseContent{}, assert.AnError)
				return m
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "validation error - user ID too long",
			userID:         invalidUserID,
			mockSetup:      func() *mockAuthManager { return &mockAuthManager{} },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error - empty user ID",
			userID:         "",
			mockSetup:      func() *mockAuthManager { return &mockAuthManager{} },
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock auth manager
			s.authManagement = tt.mockSetup()

			body, err := json.Marshal(tt.payload)
			require.NoError(t, err, "could not marshal payload")

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/admin/users/block/%s", tt.userID), bytes.NewBuffer(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()

			c := e.NewContext(req, res)
			c.SetPath("/v1/admin/users/block/:userId")
			c.SetParamNames("userId")
			c.SetParamValues(tt.userID)

			// Setup auth claims
			claims := &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{Subject: adminID},
				CustomClaims:     &auth.CustomClaims{},
			}
			if !tt.withoutPermission {
				claims.CustomClaims.(*auth.CustomClaims).Permissions = []string{string(auth.UserBlock)}
			}
			c.Set(string(auth.ClaimsContextKey), claims)

			handler := authMw.RequireAuthentication(authMw.RequirePermission(auth.UserBlock)(s.blockUserHandler))

			// Assertions
			err = handler(c)
			if he, ok := err.(*echo.HTTPError); ok {
				assert.Equal(t, tt.expectedStatus, he.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.Code)

				var actual repository.UserBlock
				err = json.NewDecoder(res.Body).Decode(&actual)

				if tt.expectedStatus != http.StatusBadRequest {
					require.NoError(t, err, "error decoding response body")
					assert.Equal(t, tt.userID, actual.UserID, "user id should be the same")
					assert.Equal(t, adminID, actual.BlockedBy, "blocked by should be the admin")
					assert.Equal(t, tt.payload.Reason, actual.Reason, "reason should be the same")
					assert.NotNil(t, actual.BlockedAt, "blocked at should not be nil")
					assert.Nil(t, actual.UnblockedBy, "unblocked by should be nil")
					assert.Nil(t, actual.UnblockedAt, "unblocked at should be nil")
					if actual.UserEmail != nil {
						assert.Equal(t, &tt.userEmail, actual.UserEmail, "user email should be the same")
					}
				}
			}
		})
	}

	t.Cleanup(cleanup)
}

func TestUnblockUserHandler(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	authMw := auth.NewMiddleware(s.cfg.Auth, s.logger)

	validUserID := "auth0|507f1f77bcf86cd799439011"
	invalidUserID := "user-id-that-is-way-too-long-and-exceeds-the-maximum-length-of-fifty-characters"

	rep := repository.New(s.db)
	_, err := rep.BlockUser(context.Background(), repository.BlockUserParams{UserID: validUserID, BlockedBy: adminID})
	require.NoError(t, err)

	tests := []struct {
		name              string
		userID            string
		withoutPermission bool
		mockSetup         func() *mockAuthManager
		expectedStatus    int
	}{
		{
			name:              "no required permission",
			userID:            validUserID,
			withoutPermission: true,
			mockSetup:         func() *mockAuthManager { return &mockAuthManager{} },
			expectedStatus:    http.StatusForbidden,
		},
		{
			name:   "successful unblock",
			userID: validUserID,
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("UnblockUser", mock.Anything, validUserID).Return(nil)
				return m
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "user block not found",
			userID: "non-existent-user-id",
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("UnblockUser", mock.Anything, validUserID).Return(nil)
				return m
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "user not found in Auth0 (404)",
			userID: validUserID,
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("UnblockUser", mock.Anything, validUserID).Return(&core.APIError{
					StatusCode: http.StatusNotFound,
				})
				return m
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "Auth0 returns 400 bad request",
			userID: validUserID,
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("UnblockUser", mock.Anything, validUserID).Return(&core.APIError{
					StatusCode: http.StatusBadRequest,
				})
				return m
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Auth0 returns 429 rate limit",
			userID: validUserID,
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("UnblockUser", mock.Anything, validUserID).Return(&core.APIError{
					StatusCode: http.StatusTooManyRequests,
				})
				return m
			},
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:   "non-API error returns 500",
			userID: validUserID,
			mockSetup: func() *mockAuthManager {
				m := &mockAuthManager{}
				m.On("UnblockUser", mock.Anything, validUserID).Return(assert.AnError)
				return m
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "validation error - user ID too long",
			userID:         invalidUserID,
			mockSetup:      func() *mockAuthManager { return &mockAuthManager{} },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error - empty user ID",
			userID:         "",
			mockSetup:      func() *mockAuthManager { return &mockAuthManager{} },
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock auth manager
			s.authManagement = tt.mockSetup()

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/admin/users/unblock/%s", tt.userID), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()

			c := e.NewContext(req, res)
			c.SetPath("/v1/admin/users/unblock/:userId")
			c.SetParamNames("userId")
			c.SetParamValues(tt.userID)

			// Setup auth claims
			claims := &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{Subject: adminID},
				CustomClaims:     &auth.CustomClaims{},
			}
			if !tt.withoutPermission {
				claims.CustomClaims.(*auth.CustomClaims).Permissions = []string{string(auth.UserUnblock)}
			}
			c.Set(string(auth.ClaimsContextKey), claims)

			handler := authMw.RequireAuthentication(authMw.RequirePermission(auth.UserUnblock)(s.unblockUserHandler))

			// Assertions
			err := handler(c)
			if he, ok := err.(*echo.HTTPError); ok {
				assert.Equal(t, tt.expectedStatus, he.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.Code)

				var actual repository.UserBlock
				err = json.NewDecoder(res.Body).Decode(&actual)

				if tt.expectedStatus != http.StatusBadRequest {
					require.NoError(t, err, "error decoding response body")
					assert.Equal(t, tt.userID, actual.UserID, "user id should be the same")
					assert.Equal(t, adminID, actual.BlockedBy, "blocked by should be the admin")
					assert.NotNil(t, actual.BlockedAt, "blocked at should not be nil")
					assert.Equal(t, &adminID, actual.UnblockedBy, "unblocked by should be the admin")
					assert.NotNil(t, actual.UnblockedAt, "unblocked at should not be nil")
				}
			}
		})
	}

	t.Cleanup(cleanup)
}

func TestGetUserBlocks(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	authMw := auth.NewMiddleware(s.cfg.Auth, s.logger)

	// Setup mock to expect BlockUser calls for test data setup
	mockAuth := &mockAuthManager{}
	mockAuth.On("BlockUser", mock.Anything, mock.Anything).Return(&management.UpdateUserResponseContent{}, nil)
	s.authManagement = mockAuth

	for i := range 15 {
		blockUser(t, s, e, fmt.Sprintf("user-%d", i), BlockUserDTO{Reason: nil})
	}

	tests := []struct {
		name              string
		withoutPermission bool
		filters           UserBlocksFilters
		expectedStatus    int
		expectedBlocks    int
	}{
		{name: "no required permission", filters: UserBlocksFilters{PaginationFilters: PaginationFilters{Page: 1, PageSize: 101}}, withoutPermission: true, expectedStatus: http.StatusForbidden},
		{name: "return all user blocks", filters: UserBlocksFilters{PaginationFilters: PaginationFilters{Page: 1, PageSize: 25}}, expectedStatus: http.StatusOK, expectedBlocks: 15},
		{name: "return user blocks for page=1 and pageSize=5", filters: UserBlocksFilters{PaginationFilters: PaginationFilters{Page: 1, PageSize: 5}}, expectedStatus: http.StatusOK, expectedBlocks: 5},
		{name: "return user blocks for page=3 and pageSize=5", filters: UserBlocksFilters{PaginationFilters: PaginationFilters{Page: 3, PageSize: 5}}, expectedStatus: http.StatusOK, expectedBlocks: 5},
		{name: "return user blocks for page=4 and pageSize=5", filters: UserBlocksFilters{PaginationFilters: PaginationFilters{Page: 4, PageSize: 5}}, expectedStatus: http.StatusOK},
		{name: "error on 0 page", filters: UserBlocksFilters{PaginationFilters: PaginationFilters{Page: 0, PageSize: 25}}, expectedStatus: http.StatusBadRequest},
		{name: "error on page > max", filters: UserBlocksFilters{PaginationFilters: PaginationFilters{Page: 20_000, PageSize: 25}}, expectedStatus: http.StatusBadRequest},
		{name: "error on 0 pageSize", filters: UserBlocksFilters{PaginationFilters: PaginationFilters{Page: 1, PageSize: 0}}, expectedStatus: http.StatusBadRequest},
		{name: "error on pageSize > max", filters: UserBlocksFilters{PaginationFilters: PaginationFilters{Page: 1, PageSize: 101}}, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/v1/admin/users/blocks?page=%d&pageSize=%d", tt.filters.Page, tt.filters.PageSize)

			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			res := httptest.NewRecorder()
			c := e.NewContext(req, res)
			c.SetPath("/v1/admin/users/blocks")

			claims := &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{Subject: adminID},
				CustomClaims:     &auth.CustomClaims{},
			}
			if !tt.withoutPermission {
				claims.CustomClaims.(*auth.CustomClaims).Permissions = []string{string(auth.GetUserBlocks)}
			}
			c.Set(string(auth.ClaimsContextKey), claims)

			handler := authMw.RequireAuthentication(authMw.RequirePermission(auth.GetUserBlocks)(s.getUserBlocks))

			// Assertions
			err := handler(c)
			if he, ok := err.(*echo.HTTPError); ok {
				assert.Equal(t, tt.expectedStatus, he.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.Code)

				var actual PaginatedUserBlocks
				err = json.NewDecoder(res.Body).Decode(&actual)
				require.NoError(t, err, "error decoding response body")
				assert.Equal(t, tt.expectedBlocks, len(actual.Items), "incorrect number of user blocks")
			}
		})
	}

	t.Cleanup(cleanup)
}

func blockUser(t *testing.T, s *Server, e *echo.Echo, userID string, payload BlockUserDTO) repository.UserBlock {
	authMw := auth.NewMiddleware(s.cfg.Auth, s.logger)
	claims := &validator.ValidatedClaims{
		RegisteredClaims: validator.RegisteredClaims{Subject: adminID},
		CustomClaims:     &auth.CustomClaims{Permissions: []string{string(auth.UserBlock)}},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err, "could not marshal payload")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/admin/users/block/%s", userID), bytes.NewBuffer(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	res := httptest.NewRecorder()

	c := e.NewContext(req, res)
	c.SetPath("/v1/admin/users/block/:userId")
	c.SetParamNames("userId")
	c.SetParamValues(userID)
	c.Set(string(auth.ClaimsContextKey), claims)

	handler := authMw.RequireAuthentication(authMw.RequirePermission(auth.UserBlock)(s.blockUserHandler))
	err = handler(c)
	require.NoError(t, err)

	var userBlock repository.UserBlock
	err = json.NewDecoder(res.Body).Decode(&userBlock)
	require.NoError(t, err, "error decoding response body")

	return userBlock
}
