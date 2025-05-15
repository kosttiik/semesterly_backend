package handlers

import (
	"net/http"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
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
// @Router /get-group-schedule/{uuid} [get]
func (a *App) GetGroupScheduleHandler(c echo.Context) error {
	uuid := c.Param("uuid")
	var scheduleItems []models.ScheduleItem

	err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Preload("Groups").
			Preload("Teachers").
			Preload("Audiences").
			Preload("Disciplines").
			Joins("JOIN schedule_item_groups ON schedule_item_groups.schedule_item_id = schedule_items.id").
			Joins("JOIN groups ON groups.id = schedule_item_groups.group_id").
			Where("groups.uuid = ?", uuid).
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
