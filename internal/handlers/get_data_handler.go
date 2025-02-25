package handlers

import (
	"net/http"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/labstack/echo/v4"
)

// GetDataHandler отправляет JSON со всем расписанием из базы данных
// @Summary Получение расписания
// @Description Возвращает данные расписания из базы данных в формате JSON
// @Tags GetData
// @Accept json
// @Produce json
// @Success 200 {array} models.ScheduleItem "Список элементов расписания"
// @Failure 500 {object} map[string]string "error: Failed to fetch schedule items"
// @Router /api/v1/get-data [get]
func (a *App) GetDataHandler(c echo.Context) error {
	var scheduleItems []models.ScheduleItem

	// Загрузка данных с подгрузкой групп, аудиторий и преподавателей
	if err := a.DB.Preload("Groups").Preload("Teachers").Preload("Audiences").Find(&scheduleItems).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch schedule items"})
	}

	return c.JSON(http.StatusOK, scheduleItems)
}
