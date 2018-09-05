package ping

import (
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	pageCollection = "pages"
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
type SinglePage struct {
	Data Page `json:"data"`
}
type PageCollection struct {
	Data []Page `json:"data"`
}

type PageRepository struct {
	Session *mgo.Session
}

// type Repository interface {
// 	// Create(*pb.Consignment) error
// 	GetAll() ([]*Page, error)
// 	Close()
// }

// Close closes the database session after each query has ran.
// Mgo creates a 'master' session on start-up, it's then good practice
// to copy a new session for each request that's made. This means that
// each request has its own database session. This is safer and more efficient,
// as under the hood each session has its own database socket and error handling.
// Using one main database socket means requests having to wait for that session.
// I.e this approach avoids locking and allows for requests to be processed concurrently. Nice!
// But... it does mean we need to ensure each session is closed on completion. Otherwise
// you'll likely build up loads of dud connections and hit a connection limit. Not nice!
func (repo *PageRepository) Close() {
	repo.Session.Close()
}

func (repo *PageRepository) collection() *mgo.Collection {
	return repo.Session.DB("").C(pageCollection)
}

func (r *PageRepository) GetAll() (pages []*Page, err error) {
	// Find normally takes a query, but as we want everything, we can nil this.
	// We then bind our consignments variable by passing it as an argument to .All().
	// That sets consignments to the result of the find query.
	// There's also a `One()` function for single results.
	err = r.collection().Find(nil).All(&pages)

	return
}

////old
func (r *PageRepository) AllPages() (PageCollection, error) {
	result := PageCollection{[]Page{}}
	err := r.collection().Find(bson.M{}).Select(bson.M{"content": 0}).All(&result.Data)

	if err != nil {
		return result, err
	}

	return result, nil
}

// func (r *PageRepository) AllPageHistory(id string) (PageEntryCollection, error) {
// 	result := PageEntryCollection{[]PageEntry{}}
// 	err := r.collection().Find(bson.M{"page": bson.ObjectIdHex(id)}).All(&result.Data)
// 	fmt.Println("Build the pagination")
// 	if err != nil {
// 		return result, err
// 	}

// 	return result, nil
// }

func (r *PageRepository) Find(id string) (SinglePage, error) {
	result := SinglePage{}
	err := r.collection().FindId(bson.ObjectIdHex(id)).One(&result.Data)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *PageRepository) Create(page *Page) error {
	result := SinglePage{}
	_ = r.collection().Find(bson.M{"url": page.Url}).One(&result.Data)

	if result.Data.Id != "" {
		panic("Page already exists")
	}

	id := bson.NewObjectId()

	if page.Url == "" {
		panic("Page cannot be created without the URL value")
	}

	_, err := r.collection().UpsertId(id, page)
	if err != nil {
		panic(err)
	}

	page.Id = id

	return nil
}

func (r *PageRepository) Update(page *Page) error {
	err := r.collection().UpdateId(page.Id, page)
	if err != nil {
		panic(err)
	}

	return nil
}

func (r *PageRepository) Delete(id string) error {
	err := r.collection().RemoveId(bson.ObjectIdHex(id))
	if err != nil {
		panic(err)
	}

	return nil
}
