package handlers

import (
	"net/http"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/labstack/echo/v4"
)

// GetGroupsHandler отправляет JSON со списком всех групп
// @Summary Получение списка групп
// @Description Возвращает данные всех групп из базы данных в формате JSON
// @Tags GetGroups
// @Accept json
// @Produce json
// @Success 200 {array} models.Group "Список групп"
// @Failure 500 {object} map[string]string "error: Failed to fetch groups"
// @Router /get-groups [get]
func (a *App) GetGroupsHandler(c echo.Context) error {
	var groups []models.Group

	if err := a.DB.Order("name").Find(&groups).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch groups"})
	}

	return c.JSON(http.StatusOK, groups)
}
