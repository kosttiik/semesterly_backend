package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
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

	if len(body) == 0 {
		return nil // Пропуск пустых ответов
	}

	// Проверка на HTML-страницу (иногда API МГТУ возвращает HTML вместо JSON)
	if strings.HasPrefix(string(body), "<") {
		return nil // Пропуск невалидных ответов
	}

	return json.Unmarshal(body, target)
}

func (a *App) InsertDataHandler(c echo.Context) error {
	var structure models.Structure
	structureURL := "https://lks.bmstu.ru/lks-back/api/v1/structure"

	err := fetchJSON(structureURL, &structure)
	if err != nil {
		log.Printf("Failed to fetch structure: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch structure"})
	}

	groupUUIDs := extractGroupUUIDs(structure.Data.Children)
	log.Printf("Fetched %d group UUIDs", len(groupUUIDs))

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]string, 0)
	sem := make(chan struct{}, 5) // Ограничение на количество одновременных горутин

	for _, uuid := range groupUUIDs {
		wg.Add(1)
		sem <- struct{}{} // Получаем токен
		go func(uuid string) {
			defer wg.Done()
			defer func() { <-sem }() // Высвобождаем токен

			var schedule models.Schedule
			var exams models.ExamResponse

			scheduleURL := "https://lks.bmstu.ru/lks-back/api/v1/schedules/groups/" + uuid + "/public"
			examURL := "https://lks.bmstu.ru/lks-back/api/v1/schedules/exams/" + uuid + "/public"

			err := fetchJSON(scheduleURL, &schedule)
			if err != nil {
				log.Printf("Failed to fetch schedule for group %s: %v", uuid, err)
				mu.Lock()
				errors = append(errors, "Failed to fetch schedule for group "+uuid)
				mu.Unlock()
				return
			}
			log.Printf("Fetched schedule for group %s", uuid)

			err = fetchJSON(examURL, &exams)
			if err != nil {
				log.Printf("Failed to fetch exams for group %s: %v", uuid, err)
				mu.Lock()
				errors = append(errors, "Failed to fetch exams for group "+uuid)
				mu.Unlock()
				return
			}
			log.Printf("Fetched exams for group %s", uuid)

			for _, item := range schedule.Data.Schedule {
				err = a.DB.Create(&item).Error
				if err != nil {
					log.Printf("Failed to insert schedule item for group %s: %v", uuid, err)
					mu.Lock()
					errors = append(errors, "Failed to insert schedule item for group "+uuid)
					mu.Unlock()
					return
				}
				log.Printf("Inserted schedule item for group %s", uuid)
			}

			for _, item := range exams.Data {
				err = a.DB.Create(&item).Error
				if err != nil {
					log.Printf("Failed to insert exam item for group %s: %v", uuid, err)
					mu.Lock()
					errors = append(errors, "Failed to insert exam item for group "+uuid)
					mu.Unlock()
					return
				}
				log.Printf("Inserted exam item for group %s", uuid)
			}

			// Задержка для снижения нагрузки на API МГТУ (возможно обход блокировки...)
			time.Sleep(5 * time.Second)
		}(uuid)
	}

	wg.Wait()

	if len(errors) > 0 {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"errors": errors})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Data inserted successfully"})
}

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
