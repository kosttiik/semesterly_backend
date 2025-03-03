package models

import (
	"encoding/json"
	"time"
)

type Schedule struct {
	Data struct {
		Type     string         `json:"type"`
		UUID     string         `json:"uuid"`
		Title    string         `json:"title"`
		Schedule []ScheduleItem `json:"schedule"`
	} `json:"data"`
	Date time.Time `json:"date"`
}

type ScheduleItem struct {
	ID          uint         `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time    `json:"-"`
	UpdatedAt   time.Time    `json:"-"`
	DeletedAt   time.Time    `json:"-" gorm:"index"`
	Day         int          `json:"day"`
	Time        int          `json:"time"`
	Week        string       `json:"week"`
	Groups      []Group      `json:"groups" gorm:"many2many:schedule_item_groups;"`
	Stream      string       `json:"stream"`
	EndTime     string       `json:"endTime"`
	Teachers    []Teacher    `json:"teachers" gorm:"many2many:schedule_item_teachers;"`
	Audiences   []Audience   `json:"audiences" gorm:"many2many:schedule_item_audiences;"`
	StartTime   string       `json:"startTime"`
	Disciplines []Discipline `json:"disciplines" gorm:"many2many:schedule_item_disciplines;"`
	// Временное поле для парсинга JSON
	DisciplineRaw Discipline `json:"discipline" gorm:"-"`
	Permission    string     `json:"permission"`
}

// Кастомная сериализация для ScheduleItem (пока что только таким методом смог убрать пустую дисциплину с id 0 в ответе)
func (s ScheduleItem) MarshalJSON() ([]byte, error) {
	type Alias ScheduleItem
	return json.Marshal(&struct {
		ID          uint         `json:"id"`
		// CreatedAt   time.Time    `json:"-"`
		// UpdatedAt   time.Time    `json:"-"`
		// DeletedAt   time.Time    `json:"-"`
		Day         int          `json:"day"`
		Time        int          `json:"time"`
		Week        string       `json:"week"`
		Groups      []Group      `json:"groups"`
		Stream      string       `json:"stream"`
		EndTime     string       `json:"endTime"`
		Teachers    []Teacher    `json:"teachers"`
		Audiences   []Audience   `json:"audiences"`
		StartTime   string       `json:"startTime"`
		Disciplines []Discipline `json:"disciplines"`
		Permission  string       `json:"permission"`
	}{
		ID:          s.ID,
		// CreatedAt:   s.CreatedAt,
		// UpdatedAt:   s.UpdatedAt,
		// DeletedAt:   s.DeletedAt,
		Day:         s.Day,
		Time:        s.Time,
		Week:        s.Week,
		Groups:      s.Groups,
		Stream:      s.Stream,
		EndTime:     s.EndTime,
		Teachers:    s.Teachers,
		Audiences:   s.Audiences,
		StartTime:   s.StartTime,
		Disciplines: s.Disciplines,
		Permission:  s.Permission,
	})
}

type Group struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	CreatedAt     time.Time `json:"-"`
	UpdatedAt     time.Time `json:"-"`
	DeletedAt     time.Time `json:"-" gorm:"index"`
	Name          string    `json:"name"`
	UUID          string    `json:"uuid"`
	DepartmentUID string    `json:"department_uid"`
}

type Teacher struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	CreatedAt  time.Time `json:"-"`
	UpdatedAt  time.Time `json:"-"`
	DeletedAt  time.Time `json:"-" gorm:"index"`
	UUID       string    `json:"uuid"`
	LastName   string    `json:"lastName"`
	FirstName  string    `json:"firstName"`
	MiddleName string    `json:"middleName"`
}

type Audience struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	CreatedAt     time.Time `json:"-"`
	UpdatedAt     time.Time `json:"-"`
	DeletedAt     time.Time `json:"-" gorm:"index"`
	Name          string    `json:"name"`
	UUID          string    `json:"uuid"`
	Building      string    `json:"building"`
	DepartmentUID *string   `json:"department_uid"` // Может быть null
}

type Discipline struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	DeletedAt time.Time `json:"-" gorm:"index"`
	Abbr      string    `json:"abbr"`
	ActType   string    `json:"actType"`
	FullName  string    `json:"fullName"`
	ShortName string    `json:"shortName"`
}
