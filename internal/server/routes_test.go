package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rousage/shortener/internal/appvalidator"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	e := echo.New()
	e.Validator = appvalidator.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	c := e.NewContext(req, resp)
	s := &Server{}

	// Assertions
	err := s.helloWorldHandler(c)
	assert.Nil(t, err, "handler() returned an error")
	assert.Equal(t, http.StatusOK, resp.Code, "handler() wrong status code")

	expected := map[string]string{"message": "Hello World"}
	var actual map[string]string
	// Decode the response body into the actual map
	if err := json.NewDecoder(resp.Body).Decode(&actual); err != nil {
		t.Errorf("handler() error decoding response body: %s", err)
		return
	}
	assert.Equal(t, expected, actual, "handler() wrong response body")
}
