package ping

import (
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	pageEntryCollection = "page_entry"
)

type IPageEntryRepository interface {
	Create(*PageEntry) error
	Close()
}

type PageEntryRepository struct {
	Session *mgo.Session
}

func (repo *PageEntryRepository) Close() {
	repo.Session.Close()
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

// unexported methods
func (repo *PageEntryRepository) collection() *mgo.Collection {
	return repo.Session.DB("").C(pageEntryCollection)
}

func (r *PageEntryRepository) Create(pageEntry *PageEntry) error {
	// result := SinglePage{}
	// _ = r.collection().Find(bson.M{"url": page.Url}).One(&result.Data)

	// if result.Data.Id != "" {
	// 	return errors.New("Page already exist")
	// }

	// id := bson.NewObjectId()

	// if page.Url == "" {
	// 	return errors.New("Page cannot be created without the URL value")
	// }

	// _, err := r.collection().UpsertId(id, page)
	// if err != nil {
	// 	return err
	// }

	// page.Id = id

	return nil
}

// func (repo *PageEntryRepository) GetAll(page *Page) (entries []*PageEntry, err error) {
// 	//page here
// 	err = repo.collection().Find(nil).All(&entries)

// 	return
// }

// PageEntryCollection is a History for a single page
// type PageEntryCollection struct {
// 	Data []PageEntry `json:"data"`
// }
