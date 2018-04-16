package models

type Page struct {
	DocumentBase `bson:",inline"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Url          string `json:"url"`
	RescueUrl    string `json:"rescue_url"`
	Interval     int    `json:"interval"`
	LastStatus   int    `json:"laststatus" bson:"laststatus"`
	Content      string `json:"content" bson:"content"`
}

type PageCollection struct {
	Data []Page `json:"data"`
}

type SinglePage struct {
	Data Page `json:"data"`
}
