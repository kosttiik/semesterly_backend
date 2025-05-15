package handlers

import (
	"log"
	"net/http"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// ClearDataHandler удаляет все данные из базы данных
// @Summary Очистка базы данных
// @Description Удаляет все данные из всех таблиц базы данных
// @Tags ClearData
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "message: Database cleared successfully"
// @Failure 500 {object} map[string]string "error: Failed to clear database"
// @Router /clear-data [post]
func (a *App) ClearDataHandler(c echo.Context) error {
	// Выполняем очистку в транзакции
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		// Сначала удаляем все ассоциации
		if err := tx.Table("schedule_item_audiences").Where("1 = 1").Unscoped().Delete(&struct{}{}).Error; err != nil {
			return err
		}
		if err := tx.Table("schedule_item_disciplines").Where("1 = 1").Unscoped().Delete(&struct{}{}).Error; err != nil {
			return err
		}
		if err := tx.Table("schedule_item_groups").Where("1 = 1").Unscoped().Delete(&struct{}{}).Error; err != nil {
			return err
		}
		if err := tx.Table("schedule_item_teachers").Where("1 = 1").Unscoped().Delete(&struct{}{}).Error; err != nil {
			return err
		}
		if err := tx.Table("exam_disciplines").Where("1 = 1").Unscoped().Delete(&struct{}{}).Error; err != nil {
			return err
		}

		// Затем удаляем основные таблицы
		tables := []any{
			&models.ScheduleItem{},
			&models.Exam{},
			&models.Teacher{},
			&models.Group{},
			&models.Audience{},
			&models.Discipline{},
		}

		for _, table := range tables {
			if err := tx.Unscoped().Where("1 = 1").Delete(table).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Failed to clear database: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to clear database",
		})
	}

	// Отправляем уведомление через WebSocket о том, что данные очищены
	a.Hub.BroadcastProgress(ProgressUpdate{
		Type:       "clearProgress",
		Message:    "Database cleared successfully",
		Percentage: 100,
	})

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Database cleared successfully",
	})
}
