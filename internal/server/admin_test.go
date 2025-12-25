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
	)

	for i := range 5 {
		createShortUrl(t, s, e, fmt.Sprintf("https://example-%d.com", i), "")
		createShortUrl(t, s, e, fmt.Sprintf("https://example-one-%d.com", i), userID_1)
		createShortUrl(t, s, e, fmt.Sprintf("https://example-two-%d.com", i), userID_2)
	}

	tests := []struct {
		name              string
		withoutPermission bool
		page              int
		pageSize          int
		expectedStatus    int
		expectedUrls      int
	}{
		{name: "no required permission", page: 1, pageSize: 101, withoutPermission: true, expectedStatus: http.StatusForbidden},
		{name: "return urls", page: 1, pageSize: 25, expectedStatus: http.StatusOK, expectedUrls: 15},
		{name: "return urls for page=1 and pageSize=5", page: 1, pageSize: 5, expectedStatus: http.StatusOK, expectedUrls: 5},
		{name: "return urls for page=3 and pageSize=5", page: 3, pageSize: 5, expectedStatus: http.StatusOK, expectedUrls: 5},
		{name: "return urls for page=4 and pageSize=5", page: 4, pageSize: 5, expectedStatus: http.StatusOK},
		{name: "error on 0 page", page: 0, pageSize: 25, expectedStatus: http.StatusBadRequest},
		{name: "error on page > max", page: 20_000, pageSize: 25, expectedStatus: http.StatusBadRequest},
		{name: "error on 0 pageSize", page: 1, pageSize: 0, expectedStatus: http.StatusBadRequest},
		{name: "error on pageSize > max", page: 1, pageSize: 101, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/admin/urls?page=%d&pageSize=%d", tt.page, tt.pageSize), nil)
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
