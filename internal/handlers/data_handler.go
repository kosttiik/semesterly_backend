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

func fetchJSON(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if len(body) == 0 || strings.HasPrefix(string(body), "<") {
		return nil // Пропуск пустых и невалидных ответов
	}

	return json.Unmarshal(body, target)
}

// Вставка данных в БД
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
	sem := make(chan struct{}, 8) // Лимит одновременно работающих горутин

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

func (a *App) processGroupData(uuid string, mu *sync.Mutex, errors *[]string) error {
	var schedule models.Schedule
	var exams models.ExamResponse

	scheduleURL := "https://lks.bmstu.ru/lks-back/api/v1/schedules/groups/" + uuid + "/public"
	examURL := "https://lks.bmstu.ru/lks-back/api/v1/schedules/exams/" + uuid + "/public"

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

func (a *App) insertToDatabase(scheduleItems []models.ScheduleItem, examItems []models.Exam, uuid string, mu *sync.Mutex, errors *[]string) error {
	for _, item := range scheduleItems {
		if err := a.DB.Create(&item).Error; err != nil {
			appendError(mu, errors, fmt.Sprintf("Failed to insert schedule item for group %s", uuid))
			return err
		}
		log.Printf("Inserted schedule item for group %s", uuid)
	}

	for _, item := range examItems {
		if err := a.DB.Create(&item).Error; err != nil {
			appendError(mu, errors, fmt.Sprintf("Failed to insert exam item for group %s", uuid))
			return err
		}
		log.Printf("Inserted exam item for group %s", uuid)
	}

	return nil
}

// Получение расписания из БД
func (a *App) GetDataHandler(c echo.Context) error {
	var scheduleItems []models.ScheduleItem
	var exams []models.Exam

	if err := a.DB.Find(&scheduleItems).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch schedule items"})
	}

	if err := a.DB.Find(&exams).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch exams"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"schedule_items": scheduleItems,
		"exams":          exams,
	})
}

// Извлечение UUID групп из структуры
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

func appendError(mu *sync.Mutex, errors *[]string, message string) {
	mu.Lock()
	defer mu.Unlock()
	*errors = append(*errors, message)
}

func (a *App) WriteScheduleToFile(c echo.Context) error {
	var scheduleItems []models.ScheduleItem
	if err := a.DB.Find(&scheduleItems).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch schedule items"})
	}

	filePath := "/usr/src/semesterly/data/schedule.csv"
	file, err := os.Create(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create file"})
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Day", "Time", "Week", "Stream", "EndTime", "StartTime", "Discipline", "Permission", "Teachers", "Audiences"})
	for _, item := range scheduleItems {
		teachers := make([]string, len(item.Teachers))
		for i, teacher := range item.Teachers {
			teachers[i] = fmt.Sprintf("%s %s %s", teacher.LastName, teacher.FirstName, teacher.MiddleName)
		}

		audiences := make([]string, len(item.Audiences))
		for i, audience := range item.Audiences {
			audiences[i] = audience.Name
		}

		writer.Write([]string{
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
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Schedule written to file successfully"})
}

func (a *App) WriteExamsToFile(c echo.Context) error {
	var exams []models.Exam
	if err := a.DB.Find(&exams).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch exams"})
	}

	filePath := "/usr/src/semesterly/data/exams.csv"
	file, err := os.Create(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create file"})
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Room", "ExamDate", "ExamTime", "LastName", "FirstName", "MiddleName", "Discipline"})
	for _, exam := range exams {
		writer.Write([]string{
			exam.Room,
			exam.ExamDate,
			exam.ExamTime,
			exam.LastName,
			exam.FirstName,
			exam.MiddleName,
			exam.Discipline,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Exams written to file successfully"})
}
