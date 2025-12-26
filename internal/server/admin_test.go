package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetURLsHandler(t *testing.T) {
	s, e, cleanup := setupTestServer(t)
	authMw := auth.NewAuthMiddleware(s.cfg.Auth, s.logger)

	var (
		adminID  = "admin-id"
		userID_1 = "user-id"
		userID_2 = "user-id-2"
		trueVal  = true
		falseVal = false
	)

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
