package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/labstack/echo/v4"
)

// fetchJSON выполняет HTTP-запрос и декодирует JSON-ответ в переданную структуру
func fetchJSON(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching URL %s: %v", url, err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return err
	}

	if len(body) == 0 || strings.HasPrefix(string(body), "<") {
		log.Printf("Empty or invalid response body from URL %s", url)
		return nil
	}

	if err := json.Unmarshal(body, target); err != nil {
		log.Printf("Error unmarshalling JSON: %v", err)
		return err
	}

	return nil
}

// InsertDataHandler обрабатывает вставку данных в базу данных
func (a *App) InsertDataHandler(c echo.Context) error {
	structureURL := "https://lks.bmstu.ru/lks-back/api/v1/structure"
	var structure models.Structure

	if err := fetchJSON(structureURL, &structure); err != nil {
		log.Printf("Failed to fetch structure: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch structure"})
	}

	groupUUIDs := extractGroupUUIDs(structure.Data.Children)
	log.Printf("Fetched %d group UUIDs", len(groupUUIDs))

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]string, 0)
	sem := make(chan struct{}, 8)

	requestInterval := time.NewTicker(500 * time.Millisecond)
	defer requestInterval.Stop()

	for _, uuid := range groupUUIDs {
		<-requestInterval.C

		sem <- struct{}{}
		wg.Add(1)
		go func(uuid string) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := a.processGroupData(uuid, &mu, &errors); err != nil {
				log.Printf("Failed to process data for group %s: %v", uuid, err)
			}
		}(uuid)
	}

	wg.Wait()

	if len(errors) > 0 {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"errors": errors})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Data inserted successfully"})
}

// processGroupData обрабатывает данные расписания и экзаменов для группы
func (a *App) processGroupData(uuid string, mu *sync.Mutex, errors *[]string) error {
	var schedule models.Schedule
	var exams models.ExamResponse

	scheduleURL := fmt.Sprintf("https://lks.bmstu.ru/lks-back/api/v1/schedules/groups/%s/public", uuid)
	examURL := fmt.Sprintf("https://lks.bmstu.ru/lks-back/api/v1/schedules/exams/%s/public", uuid)

	if err := fetchJSON(scheduleURL, &schedule); err != nil {
		appendError(mu, errors, fmt.Sprintf("Failed to fetch schedule for group %s", uuid))
		return err
	}
	log.Printf("Fetched schedule for group %s", uuid)

	if err := fetchJSON(examURL, &exams); err != nil {
		appendError(mu, errors, fmt.Sprintf("Failed to fetch exams for group %s", uuid))
		return err
	}
	log.Printf("Fetched exams for group %s", uuid)

	if err := a.insertToDatabase(schedule.Data.Schedule, exams.Data, uuid, mu, errors); err != nil {
		return err
	}

	return nil
}

// insertToDatabase вставляет данные расписания в базу данных с проверкой на дублирование
func (a *App) insertToDatabase(scheduleItems []models.ScheduleItem, examItems []models.Exam, uuid string, mu *sync.Mutex, errors *[]string) error {
	for _, item := range scheduleItems {
		if err := a.DB.Where(models.ScheduleItem{
			Day:       item.Day,
			Time:      item.Time,
			Week:      item.Week,
			Stream:    item.Stream,
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
		}).FirstOrCreate(&item).Error; err != nil {
			appendError(mu, errors, fmt.Sprintf("Failed to insert schedule item for group %s", uuid))
			return err
		}
		log.Printf("Inserted schedule item for group %s", uuid)
	}

	for _, item := range examItems {
		if err := a.DB.Where(models.Exam{
			Room:       item.Room,
			ExamDate:   item.ExamDate,
			ExamTime:   item.ExamTime,
			Discipline: item.Discipline,
		}).FirstOrCreate(&item).Error; err != nil {
			appendError(mu, errors, fmt.Sprintf("Failed to insert exam item for group %s", uuid))
			return err
		}
		log.Printf("Inserted exam item for group %s", uuid)
	}

	return nil
}

// extractGroupUUIDs извлекает UUID всех групп из структуры
func extractGroupUUIDs(children []models.Child) []string {
	var uuids []string
	for _, child := range children {
		if child.NodeType == "group" {
			uuids = append(uuids, child.UUID)
		}
		if len(child.Children) > 0 {
			uuids = append(uuids, extractGroupUUIDs(child.Children)...)
		}
	}
	return uuids
}

// appendError добавляет сообщение об ошибке в список ошибок
func appendError(mu *sync.Mutex, errors *[]string, message string) {
	mu.Lock()
	defer mu.Unlock()
	*errors = append(*errors, message)
}

// WriteScheduleToFile сохраняет расписание в CSV файл
func (a *App) WriteScheduleToFile(c echo.Context) error {
	var scheduleItems []models.ScheduleItem

	// Загрузка данных с подгрузкой аудиторий и преподавателей
	if err := a.DB.Preload("Teachers").Preload("Audiences").Find(&scheduleItems).Error; err != nil {
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
	writer.Write([]string{"Day", "Time", "Week", "Stream", "EndTime", "StartTime", "Discipline", "Permission", "Teachers", "Audiences"})

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
		})
		if err != nil {
			return fmt.Errorf("failed to write to CSV: %w", err)
		}
	}

	return nil
}

// GetData отправляет JSON со всем расписанием из базы данных
func (a *App) GetDataHandler(c echo.Context) error {
	var scheduleItems []models.ScheduleItem

	// Загрузка данных с подгрузкой аудиторий и преподавателей
	if err := a.DB.Preload("Teachers").Preload("Audiences").Find(&scheduleItems).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch schedule items"})
	}

	return c.JSON(http.StatusOK, scheduleItems)
}
