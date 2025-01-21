package handlers

import (
	"net/http"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/labstack/echo/v4"
)

// GetGroupScheduleHandler отправляет JSON с расписанием конкретной группы из базы данных
// @Summary Получение расписания группы
// @Description Возвращает данные расписания конкретной группы из базы данных в формате JSON
// @Tags GetData
// @Accept json
// @Produce json
// @Param uuid path string true "UUID группы"
// @Success 200 {array} models.ScheduleItem "Список элементов расписания"
// @Failure 500 {object} map[string]string "error: Failed to fetch schedule items"
// @Router /api/v1/get-group-schedule/{uuid} [get]
func (a *App) GetGroupScheduleHandler(c echo.Context) error {
	uuid := c.Param("uuid")
	var scheduleItems []models.ScheduleItem

	// Загрузка данных с подгрузкой аудиторий и преподавателей для конкретной группы
	if err := a.DB.Preload("Teachers").Preload("Audiences").Where("group_uuid = ?", uuid).Find(&scheduleItems).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch schedule items"})
	}

	return c.JSON(http.StatusOK, scheduleItems)
}
