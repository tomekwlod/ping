package ping

import (
	"errors"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	pageCollection = "pages"
)

// IPageRepository exposes the methods for the PageRepository
// The methods are obviously the PageRepository methods, and to use the PageRepository you need to pass the *mgo.Session to it
// I know shouldn't be using IName but in this case I have a name collision; need to resolve it later
type IPageRepository interface {
	Pages() ([]*Page, error)
	PagesForPing() ([]*Page, error)
	Find(ID string) (*SinglePage, error)
	Delete(ID string) error
	Create(*Page) error
	Update(*Page) error
	Upsert(*Page) error
	Close()
}

type PageRepository struct {
	Session *mgo.Session
}

func (repo *PageRepository) Close() {
	repo.Session.Close()
}

func (r *PageRepository) Pages() (pages []*Page, err error) {
	// Find normally takes a query, but as we want everything, we can nil this.
	// We then bind our consignments variable by passing it as an argument to .All().
	// That sets consignments to the result of the find query.
	// There's also a `One()` function for single results.
	err = r.collection().Find(nil).All(&pages)

	return
}

func (r *PageRepository) PagesForPing() (pages []*Page, err error) {
	err = r.collection().Find(bson.M{"$or": []bson.M{
		bson.M{"nextPing": bson.M{"$lte": time.Now()}},
	}}).All(&pages)

	return
}

////old
func (r *PageRepository) AllPages() (PageCollection, error) {
	result := PageCollection{[]*Page{}}
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

func (r *PageRepository) Find(id string) (*SinglePage, error) {
	result := &SinglePage{}
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
		return errors.New("Page already exist")
	}

	id := bson.NewObjectId()

	if page.Url == "" {
		return errors.New("Page cannot be created without the URL value")
	}

	_, err := r.collection().UpsertId(id, page)
	if err != nil {
		return err
	}

	page.Id = id

	return nil
}

func (r *PageRepository) Update(page *Page) error {
	err := r.collection().UpdateId(page.Id, page)
	if err != nil {
		return err
	}

	return nil
}

func (r *PageRepository) Upsert(page *Page) (err error) {
	_, err = r.collection().UpsertId(page.Id, page)

	return
}

func (r *PageRepository) Delete(id string) error {
	err := r.collection().RemoveId(bson.ObjectIdHex(id))
	if err != nil {
		return err
	}

	return nil
}

// unexported methods
func (repo *PageRepository) collection() *mgo.Collection {
	return repo.Session.DB("").C(pageCollection)
}

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
	Data []*Page `json:"data"`
}
