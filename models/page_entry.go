package models

import "gopkg.in/mgo.v2/bson"

type PageEntry struct {
	DocumentBase `bson:",inline"`
	Load         int           `json:"load"`
	Page         bson.ObjectId `json:"page" bson:"page"`
}
