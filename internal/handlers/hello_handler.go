package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	_ "github.com/kosttiik/semesterly_backend/docs" // Swagger documentation
)

type App struct {
	DB *gorm.DB
}

// helloHandler - обработчик для теста работы сервера
// @Summary Проверка подключения
// @Description Проверяет, работает ли сервер и есть ли подключение к базе данных
// @Tags Hello
// @Accept json
// @Produce json
// @Success 200 {string} string "Hello, World!"
// @Router /api/v1/hello [get]
func (a *App) HelloHandler(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World! Connected to the database successfully.")
}
