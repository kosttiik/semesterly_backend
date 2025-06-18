package handlers

import (
	"fmt"
	"log"
	"net/http"

	"slices"

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
	assocTables := []string{
		"schedule_item_audiences",
		"schedule_item_disciplines",
		"schedule_item_groups",
		"schedule_item_teachers",
		"exam_disciplines",
	}
	mainTables := []struct {
		name  string
		model any
	}{
		{"schedule_items", &models.ScheduleItem{}},
		{"exams", &models.Exam{}},
		{"teachers", &models.Teacher{}},
		{"groups", &models.Group{}},
		{"audiences", &models.Audience{}},
		{"disciplines", &models.Discipline{}},
	}

	allTables := slices.Clone(assocTables)
	for _, t := range mainTables {
		allTables = append(allTables, t.name)
	}

	totalTables := len(allTables)
	currentTable := 0

	progress := func(message string) {
		percentage := 0.0
		if totalTables > 0 {
			percentage = float64(currentTable) / float64(totalTables) * 100
		}
		a.Hub.BroadcastProgress(ProgressUpdate{
			Type:           "clearProgress",
			CurrentItem:    currentTable,
			TotalItems:     totalTables,
			CompletedItems: currentTable,
			Percentage:     percentage,
			Message:        message,
		})
	}

	progress("Начинается очистка базы данных...")

	err := a.DB.Transaction(func(tx *gorm.DB) error {
		for _, table := range allTables {
			if err := tx.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
				return err
			}
			currentTable++
			progress(fmt.Sprintf("Очищена таблица: %s...", table))
		}
		return nil
	})

	if err != nil {
		log.Printf("Не удалось очистить базу данных: %v", err)
		progress("Ошибка при очистке базы данных")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Не удалось очистить базу данных",
		})
	}

	a.Hub.BroadcastProgress(ProgressUpdate{
		Type:           "clearProgress",
		CurrentItem:    totalTables,
		TotalItems:     totalTables,
		CompletedItems: totalTables,
		Percentage:     100,
		Message:        "База данных успешно очищена",
	})

	// Закрываем все WebSocket соединения после финального сообщения
	a.Hub.CloseAll()

	return c.JSON(http.StatusOK, map[string]string{
		"message": "База данных успешно очищена",
	})
}
