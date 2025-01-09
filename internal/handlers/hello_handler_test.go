package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestHelloHandler(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create app without DB since we're just testing Hello World
	app := &App{}

	// Test
	if assert.NoError(t, app.HelloHandler(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Hello, World! Connected to the database successfully.", rec.Body.String())
	}
}
