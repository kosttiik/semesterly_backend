package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/kosttiik/semesterly_backend/internal/models"
	"github.com/kosttiik/semesterly_backend/internal/utils"
	"github.com/labstack/echo/v4"
)

// InsertGroupScheduleHandler обрабатывает вставку расписания для конкретной группы в базу данных
// @Summary Вставка расписания группы
// @Description Вставляет данные расписания и экзаменов для конкретной группы в базу данных
// @Tags InsertGroupSchedule
// @Accept json
// @Produce json
// @Param uuid path string true "UUID группы"
// @Success 200 {object} map[string]string "message: Group schedule inserted successfully"
// @Failure 500 {object} map[string]interface{} "errors: [error messages]"
// @Router /insert-group-schedule/{uuid} [post]
func (a *App) InsertGroupScheduleHandler(c echo.Context) error {
	uuid := c.Param("uuid")
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]string, 0)

	wg.Add(1)
	go func(uuid string) {
		defer wg.Done()
		if err := a.processGroupScheduleData(uuid, &mu, &errors); err != nil {
			log.Printf("Failed to process data for group %s: %v", uuid, err)
		}
	}(uuid)

	wg.Wait()

	if len(errors) > 0 {
		return c.JSON(http.StatusInternalServerError, map[string]any{"errors": errors})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Group schedule inserted successfully"})
}

// processGroupScheduleData обрабатывает данные расписания и экзаменов для группы
func (a *App) processGroupScheduleData(uuid string, mu *sync.Mutex, errors *[]string) error {
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

	if err := a.insertGroupToDatabase(schedule.Data.Schedule, exams.Data, mu, errors); err != nil {
		return err
	}

	return nil
}

// insertToDatabase вставляет данные расписания в базу данных с проверкой на дублирование
func (a *App) insertGroupToDatabase(scheduleItems []models.ScheduleItem, examItems []models.Exam, mu *sync.Mutex, errors *[]string) error {
	var insertedScheduleItems, insertedExamItems int

	for _, item := range scheduleItems {
		// Сохраняем дисциплину
		var dbDiscipline models.Discipline
		if err := a.DB.Where("abbr = ? AND act_type = ? AND full_name = ? AND short_name = ?",
			item.DisciplineRaw.Abbr, item.DisciplineRaw.ActType, item.DisciplineRaw.FullName, item.DisciplineRaw.ShortName).
			FirstOrCreate(&dbDiscipline, models.Discipline{
				Abbr:      item.DisciplineRaw.Abbr,
				ActType:   item.DisciplineRaw.ActType,
				FullName:  item.DisciplineRaw.FullName,
				ShortName: item.DisciplineRaw.ShortName,
			}).Error; err != nil {
			utils.AppendError(mu, errors, fmt.Sprintf("Failed to insert discipline: %v", err))
			continue
		}

		// Создаем элемент расписания без ассоциаций
		newItem := models.ScheduleItem{
			Day:        item.Day,
			Time:       item.Time,
			Week:       item.Week,
			Stream:     item.Stream,
			StartTime:  item.StartTime,
			EndTime:    item.EndTime,
			Permission: item.Permission,
		}

		// Ищем или создаем элемент расписания
		var existingItem models.ScheduleItem
		if err := a.DB.Where(&models.ScheduleItem{
			Day:        newItem.Day,
			Time:       newItem.Time,
			Week:       newItem.Week,
			Stream:     newItem.Stream,
			StartTime:  newItem.StartTime,
			EndTime:    newItem.EndTime,
			Permission: newItem.Permission,
		}).FirstOrCreate(&existingItem).Error; err != nil {
			utils.AppendError(mu, errors, fmt.Sprintf("Failed to insert schedule item: %v", err))
			continue
		}
		insertedScheduleItems++

		// Связываем дисциплину с элементом расписания
		if err := a.DB.Model(&existingItem).Association("Disciplines").Append(&dbDiscipline); err != nil {
			utils.AppendError(mu, errors, fmt.Sprintf("Failed to associate discipline with schedule item: %v", err))
		}

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
		// Сохраняем дисциплину
		var dbDiscipline models.Discipline
		if err := a.DB.Where("full_name = ?", item.DisciplineRaw).
			FirstOrCreate(&dbDiscipline, models.Discipline{
				FullName: item.DisciplineRaw,
			}).Error; err != nil {
			utils.AppendError(mu, errors, fmt.Sprintf("Failed to insert exam discipline: %v", err))
			continue
		}

		// Создаем экзамен
		newExam := models.Exam{
			Room:       item.Room,
			ExamDate:   item.ExamDate,
			ExamTime:   item.ExamTime,
			LastName:   item.LastName,
			FirstName:  item.FirstName,
			MiddleName: item.MiddleName,
		}

		var existingExam models.Exam
		if err := a.DB.Where(&models.Exam{
			Room:       newExam.Room,
			ExamDate:   newExam.ExamDate,
			ExamTime:   newExam.ExamTime,
			LastName:   newExam.LastName,
			FirstName:  newExam.FirstName,
			MiddleName: newExam.MiddleName,
		}).FirstOrCreate(&existingExam).Error; err != nil {
			utils.AppendError(mu, errors, fmt.Sprintf("Failed to insert exam item: %v", err))
			continue
		}
		insertedExamItems++

		// Связываем дисциплину с экзаменом
		if err := a.DB.Model(&existingExam).Association("Disciplines").Append(&dbDiscipline); err != nil {
			utils.AppendError(mu, errors, fmt.Sprintf("Failed to associate discipline with exam: %v", err))
		}
	}

	log.Printf("Inserted %d schedule items and %d exam items", insertedScheduleItems, insertedExamItems)
	return nil
}
