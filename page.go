package ping

import "time"

type Page struct {
	DocumentBase `bson:",inline"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Url          string    `json:"url"`
	RescueUrl    string    `json:"rescue_url,omitempty"`
	Interval     int       `json:"interval"`
	LastStatus   int       `json:"laststatus" bson:"laststatus"`
	Content      string    `json:"content,omitempty" bson:"content"`
	Disabled     bool      `json:"disabled,omitempty" bson:"disabled"`
	NextPing     time.Time `json:"nextPing" bson:"nextPing"`
}

type PageCollection struct {
	Data []Page `json:"data"`
}

type SinglePage struct {
	Data Page `json:"data"`
}
