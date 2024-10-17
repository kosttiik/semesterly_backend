package app

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

	app := &App{} // Или инициализируйте с необходимыми зависимостями

	// Вызываем обработчик
	if assert.NoError(t, app.helloHandler(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Hello, World! Connected to the database successfully.", rec.Body.String())
	}
}
