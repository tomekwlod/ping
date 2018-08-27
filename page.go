package ping

import (
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

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

// func pages(session *mgo.Session) ping.PageCollection {
// 	result := ping.PageCollection{[]ping.Page{}}

// 	appC := appContext{session.DB(parameters.MongoDB_Database)}
// 	repo := repository{appC.db.C("pages")}

// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	return result
// }

func Pages(session *mgo.Session, forPingingOnly bool) (PageCollection, error) {
	cnf := LoadConfig()

	// sess := session.Copy()
	// defer sess.Close()

	result := PageCollection{[]Page{}}
	collection := session.DB(cnf.MongoDB_Database).C("page")

	var err error
	if forPingingOnly {
		// only the pages ready to be checked/pinged
		err = collection.Find(bson.M{"$or": []bson.M{
			bson.M{"nextPing": bson.M{"$lte": time.Now()}},
		}}).All(&result.Data)
	} else {
		// all pages, no condition
		err = collection.Find(bson.M{}).Select(bson.M{}).All(&result.Data)
	}

	if err != nil {
		return result, err
	}

	return result, nil
}

func (p *Page) InsertPage(session *mgo.Session) error {
	cnf := LoadConfig()

	// sess := session.Copy()
	// defer sess.Close()

	collection := session.DB(cnf.MongoDB_Database).C("page")

	p.SetInsertDefaults(time.Now())
	err := collection.Insert(p)

	if err != nil {
		return err
	}

	return nil
}

func DummyPage() (p *Page) {
	p = &Page{
		Name:        "Dummy page",
		Description: "Description about dummy page",
		Url:         "http://www.dummy.page.com",
		RescueUrl:   "http://www.rescue.dummy.page.com",
		Interval:    1,
		Disabled:    false,
		LastStatus:  200,
	}
	p.SetInsertDefaults(time.Now())

	return
}
