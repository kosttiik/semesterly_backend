package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// helloHandler - обработчик для теста работы сервера
// @Summary Проверка подключения
// @Description Проверяет, работает ли сервер и есть ли подключение к базе данных
// @Tags Hello
// @Accept json
// @Produce json
// @Success 200 {string} string "Hello, World!"
// @Router /hello [get]
func (a *App) HelloHandler(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World! Connected to the database successfully.")
}
