package models

import "gorm.io/gorm"

type Schedule struct {
	Data struct {
		Type     string         `json:"type"`
		UUID     string         `json:"uuid"`
		Title    string         `json:"title"`
		Schedule []ScheduleItem `json:"schedule"`
	} `json:"data"`
}

type ScheduleItem struct {
	gorm.Model
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
	gorm.Model
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

type Teacher struct {
	gorm.Model
	UUID       string `json:"uuid"`
	LastName   string `json:"lastName"`
	FirstName  string `json:"firstName"`
	MiddleName string `json:"middleName"`
}

type Audience struct {
	gorm.Model
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

type Discipline struct {
	Abbr      string `json:"abbr"`
	ActType   string `json:"actType"`
	FullName  string `json:"fullName"`
	ShortName string `json:"shortName"`
}
