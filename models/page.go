package models

type Page struct {
	DocumentBase `bson:",inline"`
	Url          string `json:"url"`
	Interval     int    `json:"interval"`
	LastStatus   int    `bson:"laststatus" json:"laststatus"`
}

type PageCollection struct {
	Data []Page `json:"data"`
}

type SinglePage struct {
	Data Page `json:"data"`
}
