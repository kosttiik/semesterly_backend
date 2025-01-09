package models

type Schedule struct {
	Data struct {
		Type     string         `json:"type"`
		UUID     string         `json:"uuid"`
		Title    string         `json:"title"`
		Schedule []ScheduleItem `json:"schedule"`
	} `json:"data"`
}

type ScheduleItem struct {
	Day        int        `json:"day"`
	Time       int        `json:"time"`
	Week       string     `json:"week"`
	Stream     string     `json:"stream"`
	EndTime    string     `json:"endTime"`
	StartTime  string     `json:"startTime"`
	Discipline Discipline `json:"discipline" gorm:"embedded"`
	Permission string     `json:"permission"`
}

type Discipline struct {
	Abbr      string `json:"abbr"`
	ActType   string `json:"actType"`
	FullName  string `json:"fullName"`
	ShortName string `json:"shortName"`
}
