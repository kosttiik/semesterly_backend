package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/kosttiik/semesterly_backend/internal/utils"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// InsertDataHandler обрабатывает вставку данных в базу данных
// @Summary Вставка данных
// @Description Вставляет данные расписания и экзаменов в базу данных
// @Tags InsertData
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "message: Data inserted successfully"
// @Failure 500 {object} map[string]interface{} "errors: [error messages]"
// @Router /insert-data [post]
func (a *App) InsertDataHandler(c echo.Context) error {
	structureURL := "https://lks.bmstu.ru/lks-back/api/v1/structure"
	var structure models.Structure

	if err := utils.FetchJSON(structureURL, &structure); err != nil {
		log.Printf("Failed to fetch structure: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch structure"})
	}

	groupUUIDs := utils.ExtractGroupUUIDs(structure.Data.Children)
	log.Printf("Fetched %d group UUIDs", len(groupUUIDs))

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]string, 0)
	sem := make(chan struct{}, 16)                                   // Ограничение в 16 горутин одновременно
	limiter := rate.NewLimiter(rate.Every(500*time.Millisecond), 16) // 16 запросов в 500 миллисекунд

	for _, uuid := range groupUUIDs {
		sem <- struct{}{}
		wg.Add(1)
		go func(uuid string) {
			defer wg.Done()
			defer func() { <-sem }()

			// Ожидание разрешения от rate limiter
			if err := limiter.Wait(context.Background()); err != nil {
				log.Printf("Rate limiter error for group %s: %v", uuid, err)
				utils.AppendError(&mu, &errors, fmt.Sprintf("Rate limiter error for group %s: %v", uuid, err))
				return
			}

			if err := a.processGroupData(uuid, &mu, &errors); err != nil {
				log.Printf("Failed to process data for group %s: %v", uuid, err)
				utils.AppendError(&mu, &errors, fmt.Sprintf("Group %s: %v", uuid, err))
			}
		}(uuid)
	}

	wg.Wait()

	if len(errors) > 0 {
		return c.JSON(http.StatusInternalServerError, map[string]any{"errors": errors})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Data inserted successfully"})
}

func (a *App) processGroupData(uuid string, mu *sync.Mutex, errors *[]string) error {
	var schedule models.Schedule
	var exams models.ExamResponse

	scheduleURL := fmt.Sprintf("https://lks.bmstu.ru/lks-back/api/v1/schedules/groups/%s/public", uuid)
	examURL := fmt.Sprintf("https://lks.bmstu.ru/lks-back/api/v1/schedules/exams/%s/public", uuid)

	if err := utils.FetchJSON(scheduleURL, &schedule); err != nil {
		utils.AppendError(mu, errors, fmt.Sprintf("Failed to fetch schedule for group %s", uuid))
		return err
	}
	log.Printf("Fetched schedule for group %s", uuid)

	if err := utils.FetchJSON(examURL, &exams); err != nil {
		utils.AppendError(mu, errors, fmt.Sprintf("Failed to fetch exams for group %s", uuid))
		return err
	}
	log.Printf("Fetched exams for group %s", uuid)

	if err := a.insertToDatabase(schedule.Data.Schedule, exams.Data, mu, errors); err != nil {
		return err
	}

	return nil
}

func (a *App) insertToDatabase(scheduleItems []models.ScheduleItem, examItems []models.Exam, mu *sync.Mutex, errors *[]string) error {
	var insertedScheduleItems, insertedExamItems int

	for _, item := range scheduleItems {
		// Создаем элемент расписания без ассоциаций
		newItem := models.ScheduleItem{
			Day:        item.Day,
			Time:       item.Time,
			Week:       item.Week,
			Stream:     item.Stream,
			StartTime:  item.StartTime,
			EndTime:    item.EndTime,
			Discipline: item.Discipline,
			Permission: item.Permission,
		}

		// Ищем или создаем элемент расписания по уникальным полям
		var existingItem models.ScheduleItem
		if err := a.DB.Where(&models.ScheduleItem{
			Day:        newItem.Day,
			Time:       newItem.Time,
			Week:       newItem.Week,
			Stream:     newItem.Stream,
			StartTime:  newItem.StartTime,
			EndTime:    newItem.EndTime,
			Discipline: newItem.Discipline,
			Permission: newItem.Permission,
		}).FirstOrCreate(&existingItem).Error; err != nil {
			utils.AppendError(mu, errors, fmt.Sprintf("Failed to insert schedule item: %v", err))
			continue
		}
		insertedScheduleItems++

		// Ассоциация с группами
		for _, group := range item.Groups {
			var dbGroup models.Group
			if err := a.DB.Where("uuid = ?", group.UUID).FirstOrCreate(&dbGroup, models.Group{
				Name:          group.Name,
				UUID:          group.UUID,
				DepartmentUID: group.DepartmentUID,
			}).Error; err != nil {
				utils.AppendError(mu, errors, fmt.Sprintf("Failed to insert group %s: %v", group.UUID, err))
				continue
			}
			if err := a.DB.Model(&existingItem).Association("Groups").Append(&dbGroup); err != nil {
				utils.AppendError(mu, errors, fmt.Sprintf("Failed to associate group %s with schedule item: %v", group.UUID, err))
			}
		}

		// Ассоциация с преподавателями
		for _, teacher := range item.Teachers {
			var dbTeacher models.Teacher
			if err := a.DB.Where("uuid = ?", teacher.UUID).FirstOrCreate(&dbTeacher, models.Teacher{
				UUID:       teacher.UUID,
				LastName:   teacher.LastName,
				FirstName:  teacher.FirstName,
				MiddleName: teacher.MiddleName,
			}).Error; err != nil {
				utils.AppendError(mu, errors, fmt.Sprintf("Failed to insert teacher %s: %v", teacher.UUID, err))
				continue
			}
			if err := a.DB.Model(&existingItem).Association("Teachers").Append(&dbTeacher); err != nil {
				utils.AppendError(mu, errors, fmt.Sprintf("Failed to associate teacher %s with schedule item: %v", teacher.UUID, err))
			}
		}

		// Ассоциация с аудиториями
		for _, audience := range item.Audiences {
			var dbAudience models.Audience
			if err := a.DB.Where("uuid = ?", audience.UUID).FirstOrCreate(&dbAudience, models.Audience{
				Name:          audience.Name,
				UUID:          audience.UUID,
				Building:      audience.Building,
				DepartmentUID: audience.DepartmentUID,
			}).Error; err != nil {
				utils.AppendError(mu, errors, fmt.Sprintf("Failed to insert audience %s: %v", audience.UUID, err))
				continue
			}
			if err := a.DB.Model(&existingItem).Association("Audiences").Append(&dbAudience); err != nil {
				utils.AppendError(mu, errors, fmt.Sprintf("Failed to associate audience %s with schedule item: %v", audience.UUID, err))
			}
		}
	}

	for _, item := range examItems {
		if err := a.DB.Where(models.Exam{
			Room:       item.Room,
			ExamDate:   item.ExamDate,
			ExamTime:   item.ExamTime,
			Discipline: item.Discipline,
		}).FirstOrCreate(&item).Error; err != nil {
			utils.AppendError(mu, errors, fmt.Sprintf("Failed to insert exam item: %v", err))
			continue
		}
		insertedExamItems++
	}

	log.Printf("Inserted %d schedule items and %d exam items", insertedScheduleItems, insertedExamItems)
	return nil
}
