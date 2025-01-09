package models

type Exam struct {
	Room       string `json:"room"`
	ExamDate   string `json:"examDate"`
	ExamTime   string `json:"examTime"`
	LastName   string `json:"lastName"`
	FirstName  string `json:"firstName"`
	Discipline string `json:"discipline"`
	MiddleName string `json:"middleName"`
}

type ExamResponse struct {
	Data []Exam `json:"data"`
	Date string `json:"date"`
}
