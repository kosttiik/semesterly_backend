package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestHelloHandler(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hello", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Создание приложения без подключения к БД (для теста)
	app := &App{}

	// Тест обработчика HelloHandler
	if assert.NoError(t, app.HelloHandler(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Hello, World! Connected to the database successfully.", rec.Body.String())
	}
}
