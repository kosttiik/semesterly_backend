package handlers

import (
	"net/http"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/labstack/echo/v4"
)

// GetData отправляет JSON со всем расписанием из базы данных
func (a *App) GetDataHandler(c echo.Context) error {
	var scheduleItems []models.ScheduleItem

	// Загрузка данных с подгрузкой аудиторий и преподавателей
	if err := a.DB.Preload("Teachers").Preload("Audiences").Find(&scheduleItems).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch schedule items"})
	}

	return c.JSON(http.StatusOK, scheduleItems)
}
