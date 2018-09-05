package ping

import (
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	pageEntryCollection = "page_entry"
)

type PageEntryRepository struct {
	session *mgo.Session
}

type PageEntry struct {
	DocumentBase `bson:",inline"`
	Load         float64       `json:"load"`
	Code         int           `json:"code"`
	Page         bson.ObjectId `json:"page" bson:"page"`
}
type PageEntryCollection struct {
	Data []Page `json:"data"`
}

func (repo *PageEntryRepository) collection() *mgo.Collection {
	return repo.session.DB(DBName).C(pageCollection)
}

func (repo *PageEntryRepository) GetAll(page *Page) (entries []*PageEntry, err error) {
	//page here
	err = repo.collection().Find(nil).All(&entries)

	return
}

// PageEntryCollection is a History for a single page
// type PageEntryCollection struct {
// 	Data []PageEntry `json:"data"`
// }
