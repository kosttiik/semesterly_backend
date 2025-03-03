package models

import "time"

type Exam struct {
	ID          uint         `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time    `json:"-"`
	UpdatedAt   time.Time    `json:"-"`
	DeletedAt   time.Time    `json:"-" gorm:"index"`
	Room        string       `json:"room"`
	ExamDate    string       `json:"examDate"`
	ExamTime    string       `json:"examTime"`
	LastName    string       `json:"lastName"`
	FirstName   string       `json:"firstName"`
	MiddleName  string       `json:"middleName"`
	Disciplines []Discipline `json:"disciplines" gorm:"many2many:exam_disciplines;"`
	// Временное поле для парсинга JSON
	DisciplineRaw string `json:"discipline" gorm:"-"`
}

type ExamResponse struct {
	Data []Exam `json:"data"`
	Date string `json:"date"`
}
