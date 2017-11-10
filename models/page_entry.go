package models

import "gopkg.in/mgo.v2/bson"

type PageEntry struct {
	DocumentBase `bson:",inline"`
	Load         float64       `json:"load"`
	Code         int           `json:"code"`
	Page         bson.ObjectId `json:"page" bson:"page"`
	Content      string        `json:"content" bson:"content"`
}

// PageEntryCollection is a History for a single page
type PageEntryCollection struct {
	Data []PageEntry `json:"data"`
}
