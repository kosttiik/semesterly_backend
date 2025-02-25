package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/labstack/echo/v4"
)

// WriteScheduleToFileHandler сохраняет расписание в CSV файл
// @Summary Сохранение расписания
// @Description Сохраняет данные расписания в CSV файл
// @Tags WriteSchedule
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "message: Schedule written to file successfully"
// @Failure 500 {object} map[string]string "error: Failed to fetch schedule items" "error: Failed to write schedule to file"
// @Router /write-schedule [post]
func (a *App) WriteScheduleToFileHandler(c echo.Context) error {
	var scheduleItems []models.ScheduleItem

	// Загрузка данных с подгрузкой групп, аудиторий и преподавателей
	if err := a.DB.Preload("Groups").Preload("Teachers").Preload("Audiences").Find(&scheduleItems).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch schedule items"})
	}

	filePath := "/usr/src/semesterly/data/schedule.csv"
	if err := writeToCSV(filePath, scheduleItems); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to write schedule to file"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Schedule written to file successfully"})
}

// writeToCSV записывает данные расписания в CSV файл
func writeToCSV(filePath string, scheduleItems []models.ScheduleItem) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Заголовки CSV
	writer.Write([]string{"Day", "Time", "Week", "Stream", "EndTime", "StartTime", "Discipline", "Permission", "Teachers", "Audiences", "Groups"})

	for _, item := range scheduleItems {
		// Форматирование списка преподавателей
		teachers := make([]string, len(item.Teachers))
		for i, teacher := range item.Teachers {
			teachers[i] = fmt.Sprintf("%s %s %s", teacher.LastName, teacher.FirstName, teacher.MiddleName)
		}

		// Форматирование списка аудиторий
		audiences := make([]string, len(item.Audiences))
		for i, audience := range item.Audiences {
			audiences[i] = audience.Name
		}

		// Форматирование списка групп
		groups := make([]string, len(item.Groups))
		for i, group := range item.Groups {
			groups[i] = group.Name
		}

		// Запись строки в CSV
		err := writer.Write([]string{
			fmt.Sprintf("%d", item.Day),
			fmt.Sprintf("%d", item.Time),
			item.Week,
			item.Stream,
			item.EndTime,
			item.StartTime,
			item.Discipline.FullName,
			item.Permission,
			strings.Join(teachers, "; "),
			strings.Join(audiences, "; "),
			strings.Join(groups, "; "),
		})
		if err != nil {
			return fmt.Errorf("failed to write to CSV: %w", err)
		}
	}

	return nil
}
