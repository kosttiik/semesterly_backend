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
	Abbr       string  `json:"abbr"`
	Name       string  `json:"name"`
	UUID       string  `json:"uuid"`
	NodeType   *string `json:"nodeType,omitempty"`
	Course     *int    `json:"course,omitempty"`
	Semester   *int    `json:"semester,omitempty"`
	ParentUUID *string `json:"parentUuid,omitempty"`
	Children   []Child `json:"children"`
}
