package handlers

import (
	"net/http"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// GetDataHandler отправляет JSON со всем расписанием из базы данных
// @Summary Получение расписания
// @Description Возвращает данные расписания из базы данных в формате JSON
// @Tags GetData
// @Accept json
// @Produce json
// @Success 200 {array} models.ScheduleItem "Список элементов расписания"
// @Failure 500 {object} map[string]string "error: Failed to fetch schedule items"
// @Router /get-data [get]
func (a *App) GetDataHandler(c echo.Context) error {
	var scheduleItems []models.ScheduleItem

	err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Preload("Groups").
			Preload("Teachers").
			Preload("Audiences").
			Preload("Disciplines").
			Find(&scheduleItems).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch schedule items"})
	}

	return c.JSON(http.StatusOK, scheduleItems)
}
