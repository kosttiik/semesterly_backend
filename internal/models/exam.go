package models

type Exam struct {
	Room       string `json:"room"`
	ExamDate   string `json:"examDate"`
	ExamTime   string `json:"examTime"`
	LastName   string `json:"lastName"`
	FirstName  string `json:"firstName"`
	MiddleName string `json:"middleName"`
	Discipline string `json:"discipline"`
}

type ExamResponse struct {
	Data []Exam `json:"data"`
	Date string `json:"date"`
}
