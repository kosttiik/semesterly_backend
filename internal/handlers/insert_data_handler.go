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
	"gorm.io/gorm"
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

	totalItems := len(groupUUIDs)
	if totalItems == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No groups found"})
	}

	completed := 0
	startTime := time.Now()

	var mu sync.Mutex

	// Отправляем начальное состояние прогресса
	mu.Lock()
	a.Hub.BroadcastProgress(ProgressUpdate{
		Type:           "insertProgress",
		CurrentItem:    0,
		TotalItems:     totalItems,
		CompletedItems: 0,
		Percentage:     0,
		ETA:            "Calculating...",
	})
	mu.Unlock()

	var wg sync.WaitGroup
	errors := make([]string, 0)
	sem := make(chan struct{}, 10)                                   // Ограничение в 10 горутин
	limiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 10) // 10 запросов в 100 миллисекунд

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

			mu.Lock()
			completed++
			elapsed := time.Since(startTime)
			itemsPerSecond := float64(completed) / elapsed.Seconds()
			remainingItems := totalItems - completed
			eta := time.Duration(float64(remainingItems)/itemsPerSecond) * time.Second

			// Отправляем состояние прогресса
			a.Hub.BroadcastProgress(ProgressUpdate{
				Type:           "insertProgress",
				CurrentItem:    completed,
				TotalItems:     totalItems,
				CompletedItems: completed,
				Percentage:     float64(completed) / float64(totalItems) * 100,
				ETA:            eta.Round(time.Second).String(),
			})
			mu.Unlock()
		}(uuid)
	}

	wg.Wait()

	// Финальное состояние прогресса
	a.Hub.BroadcastProgress(ProgressUpdate{
		Type:           "insertProgress",
		CurrentItem:    totalItems,
		TotalItems:     totalItems,
		CompletedItems: totalItems,
		Percentage:     100,
		ETA:            "0s",
	})

	if len(errors) > 0 {
		if completed == 0 {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error":   "All groups failed to process",
				"details": fmt.Sprintf("%v", errors),
			})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message":   "Partially completed with errors",
			"completed": completed,
			"total":     totalItems,
			"errors":    errors,
		})
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

	if err := utils.FetchJSON(examURL, &exams); err != nil {
		utils.AppendError(mu, errors, fmt.Sprintf("Failed to fetch exams for group %s", uuid))
		return err
	}

	// Получаем существующие записи расписания для сравнения
	var existingSchedules []models.ScheduleItem
	if err := a.DB.
		Preload("Groups", "groups.uuid = ?", uuid).
		Preload("Teachers").
		Preload("Audiences").
		Preload("Disciplines").
		Joins("JOIN schedule_item_groups ON schedule_item_groups.schedule_item_id = schedule_items.id").
		Joins("JOIN groups ON groups.id = schedule_item_groups.group_id").
		Where("groups.uuid = ?", uuid).
		Find(&existingSchedules).Error; err != nil {
		utils.AppendError(mu, errors, fmt.Sprintf("Failed to fetch existing schedules for group %s", err))
		return err
	}

	// Получаем существующие записи экзаменов для сравнения
	var existingExams []models.Exam
	if len(exams.Data) > 0 {
		if err := a.DB.
			Preload("Disciplines").
			Where("room = ? AND last_name = ? AND first_name = ? AND middle_name = ?",
				exams.Data[0].Room, exams.Data[0].LastName, exams.Data[0].FirstName, exams.Data[0].MiddleName).
			Find(&existingExams).Error; err != nil {
			utils.AppendError(mu, errors, fmt.Sprintf("Failed to fetch existing exams for group %s", err))
			return err
		}
	}

	// Сравниваем расписание и экзамены с существующими данными
	changes := false
	if !compareSchedules(existingSchedules, schedule.Data.Schedule) {
		changes = true

		if err := a.DB.Transaction(func(tx *gorm.DB) error {
			var existingItems []models.ScheduleItem
			if err := tx.
				Preload("Groups", "groups.uuid = ?", uuid).
				Joins("JOIN schedule_item_groups ON schedule_items.id = schedule_item_groups.schedule_item_id").
				Joins("JOIN groups ON groups.id = schedule_item_groups.group_id").
				Where("groups.uuid = ?", uuid).
				Find(&existingItems).Error; err != nil {
				return err
			}

			for _, item := range existingItems {
				// Удаляем ассоциации с группами, преподавателями, аудиториями и дисциплинами
				if err := tx.Model(&item).Association("Disciplines").Clear(); err != nil {
					return err
				}
				if err := tx.Model(&item).Association("Teachers").Clear(); err != nil {
					return err
				}
				if err := tx.Model(&item).Association("Audiences").Clear(); err != nil {
					return err
				}
				if err := tx.Model(&item).Association("Groups").Clear(); err != nil {
					return err
				}

				// Удаляем элемент расписания
				if err := tx.Delete(&item).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			utils.AppendError(mu, errors, fmt.Sprintf("Failed to clean up old schedule items for group %s: %v", uuid, err))
			return err
		}

		// Вставляем новые данные
		if err := a.insertToDatabase(schedule.Data.Schedule, exams.Data, mu, errors); err != nil {
			return err
		}
	} else {
		// Проверяем экзамены
		if !compareExams(existingExams, exams.Data) {
			changes = true
			// Обновляем только экзамены
			if err := a.insertExamsToDatabase(exams.Data, mu, errors); err != nil {
				return err
			}
		}
	}

	if changes {
		log.Printf("Updated data for group %s - found changes", uuid)
	} else {
		log.Printf("No changes needed for group %s", uuid)
	}

	return nil
}

// compareSchedules сравнивает два слайса элементов расписания
func compareSchedules(existing []models.ScheduleItem, new []models.ScheduleItem) bool {
	if len(existing) != len(new) {
		return false
	}

	// Создаем map для быстрого поиска
	existingMap := make(map[string]models.ScheduleItem)
	for _, item := range existing {
		key := fmt.Sprintf("%d-%d-%s-%s-%s-%s",
			item.Day, item.Time, item.Week, item.StartTime, item.EndTime, item.Stream)
		existingMap[key] = item
	}

	// Сравниваем каждый новый элемент
	for _, item := range new {
		key := fmt.Sprintf("%d-%d-%s-%s-%s-%s",
			item.Day, item.Time, item.Week, item.StartTime, item.EndTime, item.Stream)

		existingItem, exists := existingMap[key]
		if !exists {
			return false
		}

		// Сравниваем группы, преподавателей, аудитории и дисциплины
		if !compareGroups(existingItem.Groups, item.Groups) ||
			!compareTeachers(existingItem.Teachers, item.Teachers) ||
			!compareAudiences(existingItem.Audiences, item.Audiences) ||
			!compareDisciplines(existingItem.Disciplines, []models.Discipline{item.DisciplineRaw}) {
			return false
		}
	}

	return true
}

func compareGroups(a, b []models.Group) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]bool)
	for _, g := range a {
		aMap[g.UUID] = true
	}
	for _, g := range b {
		if !aMap[g.UUID] {
			return false
		}
	}
	return true
}

func compareTeachers(a, b []models.Teacher) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]bool)
	for _, t := range a {
		aMap[t.UUID] = true
	}
	for _, t := range b {
		if !aMap[t.UUID] {
			return false
		}
	}
	return true
}

func compareAudiences(a, b []models.Audience) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]bool)
	for _, aud := range a {
		aMap[aud.UUID] = true
	}
	for _, aud := range b {
		if !aMap[aud.UUID] {
			return false
		}
	}
	return true
}

func compareDisciplines(a []models.Discipline, b []models.Discipline) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]bool)
	for _, d := range a {
		aMap[d.FullName] = true
	}
	for _, d := range b {
		if !aMap[d.FullName] {
			return false
		}
	}
	return true
}

func compareExams(existing []models.Exam, new []models.Exam) bool {
	if len(existing) != len(new) {
		return false
	}

	existingMap := make(map[string]bool)
	for _, exam := range existing {
		key := fmt.Sprintf("%s-%s-%s-%s-%s-%s",
			exam.Room, exam.ExamDate, exam.ExamTime,
			exam.LastName, exam.FirstName, exam.MiddleName)
		existingMap[key] = true
	}

	for _, exam := range new {
		key := fmt.Sprintf("%s-%s-%s-%s-%s-%s",
			exam.Room, exam.ExamDate, exam.ExamTime,
			exam.LastName, exam.FirstName, exam.MiddleName)
		if !existingMap[key] {
			return false
		}
	}

	return true
}

func (a *App) insertExamsToDatabase(examItems []models.Exam, mu *sync.Mutex, errors *[]string) error {
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

		// Связываем дисциплину с экзаменом
		if err := a.DB.Model(&existingExam).Association("Disciplines").Append(&dbDiscipline); err != nil {
			utils.AppendError(mu, errors, fmt.Sprintf("Failed to associate discipline with exam: %v", err))
		}
	}

	return nil
}

func (a *App) insertToDatabase(scheduleItems []models.ScheduleItem, examItems []models.Exam, mu *sync.Mutex, errors *[]string) error {
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
