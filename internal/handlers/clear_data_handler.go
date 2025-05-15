package handlers

import (
	"fmt"
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
	// Список всех ассоциативных таблиц и основных таблиц
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

	// Подсчитываем общее количество строк для удаления
	totalRows := int64(0)
	tableRowCounts := make(map[string]int64)

	// Считаем строки в ассоциативных таблицах
	for _, table := range assocTables {
		var count int64
		a.DB.Table(table).Count(&count)
		tableRowCounts[table] = count
		totalRows += count
	}
	// Считаем строки в основных таблицах
	for _, t := range mainTables {
		var count int64
		a.DB.Model(t.model).Count(&count)
		tableRowCounts[t.name] = count
		totalRows += count
	}

	deletedRows := int64(0)
	// Функция для отправки прогресса через WebSocket
	progress := func(message string) {
		percentage := 0.0
		if totalRows > 0 {
			percentage = float64(deletedRows) / float64(totalRows) * 100
		}
		a.Hub.BroadcastProgress(ProgressUpdate{
			Type:           "clearProgress",
			CurrentItem:    int(deletedRows),
			TotalItems:     int(totalRows),
			CompletedItems: int(deletedRows),
			Percentage:     percentage,
			Message:        message,
		})
	}

	// Отправляем начальное состояние прогресса
	progress("Начинается очистка базы данных...")

	err := a.DB.Transaction(func(tx *gorm.DB) error {
		// Удаляем ассоциативные таблицы с прогрессом
		for _, table := range assocTables {
			if tableRowCounts[table] > 0 {
				if err := tx.Table(table).Where("1 = 1").Unscoped().Delete(&struct{}{}).Error; err != nil {
					return err
				}
				deletedRows += tableRowCounts[table]
				progress(fmt.Sprintf("Очищена ассоциативная таблица: %s...", table))
			}
		}
		// Удаляем основные таблицы с прогрессом
		for _, t := range mainTables {
			if tableRowCounts[t.name] > 0 {
				if err := tx.Unscoped().Where("1 = 1").Delete(t.model).Error; err != nil {
					return err
				}
				deletedRows += tableRowCounts[t.name]
				progress(fmt.Sprintf("Очищена основная таблица: %s...", t.name))
			}
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

	// Финальное состояние прогресса
	a.Hub.BroadcastProgress(ProgressUpdate{
		Type:           "clearProgress",
		CurrentItem:    int(totalRows),
		TotalItems:     int(totalRows),
		CompletedItems: int(totalRows),
		Percentage:     100,
		Message:        "База данных успешно очищена",
	})

	return c.JSON(http.StatusOK, map[string]string{
		"message": "База данных успешно очищена",
	})
}
