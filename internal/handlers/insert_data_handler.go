package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
// @Summary Вставка данных
// @Description Вставляет данные расписания и экзаменов в базу данных
// @Tags InsertData
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "message: Data inserted successfully"
// @Failure 500 {object} map[string]interface{} "errors: [error messages]"
// @Router /api/v1/insert-data [post]
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
	var insertedScheduleItems, insertedExamItems int

	for _, item := range scheduleItems {
		for i := range item.Groups {
			item.Groups[i].UUID = uuid
		}

		if err := a.DB.Where(models.ScheduleItem{
			Day:       item.Day,
			Time:      item.Time,
			Week:      item.Week,
			Stream:    item.Stream,
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			GroupUUID: uuid,
		}).FirstOrCreate(&item).Error; err != nil {
			appendError(mu, errors, fmt.Sprintf("Failed to insert schedule item for group %s", uuid))
			return err
		}
		insertedScheduleItems++
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
		insertedExamItems++
	}

	log.Printf("Inserted %d schedule items and %d exam items for group %s", insertedScheduleItems, insertedExamItems, uuid)
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
