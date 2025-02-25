package models

import (
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
	ID         uint       `json:"id" gorm:"primarykey"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  time.Time  `json:"deleted_at,omitempty" gorm:"index"`
	Day        int        `json:"day"`
	Time       int        `json:"time"`
	Week       string     `json:"week"`
	Groups     []Group    `json:"groups" gorm:"many2many:schedule_item_groups;"`
	Stream     string     `json:"stream"`
	EndTime    string     `json:"endTime"`
	Teachers   []Teacher  `json:"teachers" gorm:"many2many:schedule_item_teachers;"`
	Audiences  []Audience `json:"audiences" gorm:"many2many:schedule_item_audiences;"`
	StartTime  string     `json:"startTime"`
	Discipline Discipline `json:"discipline" gorm:"embedded"`
	Permission string     `json:"permission"`
}

type Group struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	DeletedAt     time.Time `json:"deleted_at,omitempty" gorm:"index"`
	Name          string    `json:"name"`
	UUID          string    `json:"uuid"`
	DepartmentUID string    `json:"department_uid"`
}

type Teacher struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeletedAt  time.Time `json:"deleted_at,omitempty" gorm:"index"`
	UUID       string    `json:"uuid"`
	LastName   string    `json:"lastName"`
	FirstName  string    `json:"firstName"`
	MiddleName string    `json:"middleName"`
}

type Audience struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	DeletedAt     time.Time `json:"deleted_at,omitempty" gorm:"index"`
	Name          string    `json:"name"`
	UUID          string    `json:"uuid"`
	Building      string    `json:"building"`
	DepartmentUID *string   `json:"department_uid"` // Может быть null
}

type Discipline struct {
	Abbr      string `json:"abbr"`
	ActType   string `json:"actType"`
	FullName  string `json:"fullName"`
	ShortName string `json:"shortName"`
}
