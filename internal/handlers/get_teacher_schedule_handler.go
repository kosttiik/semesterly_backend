package handlers

import (
	"net/http"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// GetTeacherScheduleHandler отправляет JSON с расписанием конкретного преподавателя из базы данных
// @Summary Получение расписания преподавателя
// @Description Возвращает данные расписания конкретного преподавателя из базы данных в формате JSON
// @Tags GetData
// @Accept json
// @Produce json
// @Param uuid path string true "UUID преподавателя"
// @Success 200 {array} models.ScheduleItem "Список элементов расписания"
// @Failure 500 {object} map[string]string "error: Failed to fetch schedule items"
// @Router /get-teacher-schedule/{uuid} [get]
func (a *App) GetTeacherScheduleHandler(c echo.Context) error {
	uuid := c.Param("uuid")
	var scheduleItems []models.ScheduleItem

	err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Preload("Groups").
			Preload("Teachers").
			Preload("Audiences").
			Preload("Disciplines").
			Joins("JOIN schedule_item_teachers ON schedule_item_teachers.schedule_item_id = schedule_items.id").
			Joins("JOIN teachers ON teachers.id = schedule_item_teachers.teacher_id").
			Where("teachers.uuid = ?", uuid).
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
