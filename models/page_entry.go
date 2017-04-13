package models

import "gopkg.in/mgo.v2/bson"
import "time"

type PageEntry struct {
	DocumentBase `bson:",inline"`
	Load         time.Duration `json:"load"`
	Code         int           `json:"code"`
	Page         bson.ObjectId `json:"page" bson:"page"`
}
