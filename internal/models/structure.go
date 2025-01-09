package models

type Structure struct {
	Data struct {
		Abbr     string  `json:"abbr"`
		Name     string  `json:"name"`
		UUID     string  `json:"uuid"`
		Children []Child `json:"children"`
	} `json:"data"`
}

type Child struct {
	Abbr     string  `json:"abbr"`
	Name     string  `json:"name"`
	UUID     string  `json:"uuid"`
	NodeType string  `json:"nodeType"`
	Course   int     `json:"course"`
	Semester int     `json:"semester"`
	Children []Child `json:"children"`
}
