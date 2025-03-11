package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// PingHandler - обработчик для теста работы сервера
// @Summary Проверка подключения
// @Description Проверяет, работает ли сервер и есть ли подключение к базе данных
// @Tags Ping
// @Accept json
// @Produce json
// @Success 200 {string} string "Pong! Connected to the database successfully."
// @Router /ping [get]
func (a *App) PingHandler(c echo.Context) error {
	return c.String(http.StatusOK, "Pong! Connected to the database successfully.")
}
