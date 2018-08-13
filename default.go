package ping

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type DocumentBase struct {
	Id       bson.ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	Created  time.Time     `bson:"_created" json:"_created"`
	Modified time.Time     `bson:"_modified" json:"_modified"`
}

func (d *DocumentBase) SetInsertDefaults(t time.Time) {
	d.Created = t
	d.Modified = t
}
func (d *DocumentBase) SetUpdateDefaults(t time.Time) {
	d.Modified = t
}
