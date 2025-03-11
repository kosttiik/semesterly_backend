package handlers

import (
	"net/http"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/labstack/echo/v4"
)

// GetTeachersHandler отправляет JSON со списком всех преподавателей
// @Summary Получение списка преподавателей
// @Description Возвращает данные всех преподавателей из базы данных в формате JSON
// @Tags GetTeachers
// @Accept json
// @Produce json
// @Success 200 {array} models.Teacher "Список преподавателей"
// @Failure 500 {object} map[string]string "error: Failed to fetch teachers"
// @Router /get-teachers [get]
func (a *App) GetTeachersHandler(c echo.Context) error {
	var teachers []models.Teacher

	if err := a.DB.Order("last_name").Order("first_name").Order("middle_name").Find(&teachers).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch teachers"})
	}

	return c.JSON(http.StatusOK, teachers)
}
