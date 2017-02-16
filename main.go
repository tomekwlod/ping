package main

import (
	"encoding/json"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Repo

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

type Page struct {
	DocumentBase `bson:",inline"`
	Url          string `json:"url"`
	Interval     int    `json:"interval"`
	Status       bool   `json:"status"`
}

type PageEntry struct {
	DocumentBase `bson:",inline"`
	Load         int           `json:"load"`
	Page         bson.ObjectId `json:"page" bson:"page"`
}

type PageCollection struct {
	Data []Page `json:"data"`
}

type PageResource struct {
	Data Page `json:"data"`
}

type Repository struct {
	coll *mgo.Collection
}

func (r *Repository) All() (PageCollection, error) {
	result := PageCollection{[]Page{}}
	err := r.coll.Find(nil).All(&result.Data)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *Repository) Find(id string) (PageResource, error) {
	result := PageResource{}
	err := r.coll.FindId(bson.ObjectIdHex(id)).One(&result.Data)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *Repository) Create(page *Page) error {
	id := bson.NewObjectId()
	_, err := r.coll.UpsertId(id, page)
	if err != nil {
		return err
	}

	page.Id = id

	return nil
}

func (r *Repository) Update(page *Page) error {
	err := r.coll.UpdateId(page.Id, page)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) Delete(id string) error {
	err := r.coll.RemoveId(bson.ObjectIdHex(id))
	if err != nil {
		return err
	}

	return nil
}

// Errors

type Errors struct {
	Errors []*Error `json:"errors"`
}

type Error struct {
	Id     string `json:"id"`
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func WriteError(w http.ResponseWriter, err *Error) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(Errors{[]*Error{err}})
}

var (
	mgoSession              *mgo.Session
	dbName                  = "ping"
	ErrBadRequest           = &Error{"bad_request", 400, "Bad request", "Request body is not well-formed. It must be JSON."}
	ErrNotAcceptable        = &Error{"not_acceptable", 406, "Not Acceptable", "Accept header must be set to 'application/vnd.api+json'."}
	ErrUnsupportedMediaType = &Error{"unsupported_media_type", 415, "Unsupported Media Type", "Content-Type header must be set to: 'application/vnd.api+json'."}
	ErrInternalServer       = &Error{"internal_server_error", 500, "Internal Server Error", "Something went wrong."}
)

// Middlewares

func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %+v", err)
				WriteError(w, ErrInternalServer)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		log.Printf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

func acceptHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.api+json" {
			WriteError(w, ErrNotAcceptable)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func contentTypeHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/vnd.api+json" {
			WriteError(w, ErrUnsupportedMediaType)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func bodyHandler(v interface{}) func(http.Handler) http.Handler {
	t := reflect.TypeOf(v)

	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			val := reflect.New(t).Interface()
			err := json.NewDecoder(r.Body).Decode(val)

			if err != nil {
				WriteError(w, ErrBadRequest)
				return
			}

			if next != nil {
				context.Set(r, "body", val)
				next.ServeHTTP(w, r)
			}
		}

		return http.HandlerFunc(fn)
	}

	return m
}

// Main handlers

type appContext struct {
	db *mgo.Database
}

func (c *appContext) pagesHandler(w http.ResponseWriter, r *http.Request) {
	repo := Repository{c.db.C("pages")}
	pages, err := repo.All()
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(pages)
}

func (c *appContext) pageHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	repo := Repository{c.db.C("pages")}
	page, err := repo.Find(params.ByName("id"))
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(page)
}

func (c *appContext) createpageHandler(w http.ResponseWriter, r *http.Request) {
	repo := Repository{c.db.C("pages")}

	body := context.Get(r, "body").(*PageResource)
	body.Data.SetInsertDefaults(time.Now())

	err := repo.Create(&body.Data)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(body)
}

func (c *appContext) updatepageHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	repo := Repository{c.db.C("pages")}

	body := context.Get(r, "body").(*PageResource)
	body.Data.Id = bson.ObjectIdHex(params.ByName("id"))
	body.Data.SetUpdateDefaults(time.Now())

	err := repo.Update(&body.Data)
	if err != nil {
		panic(err)
	}

	w.WriteHeader(204)
	w.Write([]byte("\n"))
}

func (c *appContext) deletepageHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	repo := Repository{c.db.C("pages")}
	err := repo.Delete(params.ByName("id"))
	if err != nil {
		panic(err)
	}

	w.WriteHeader(204)
	w.Write([]byte("\n"))
}

// Router

type router struct {
	*httprouter.Router
}

func (r *router) Get(path string, handler http.Handler) {
	r.GET(path, wrapHandler(handler))
}

func (r *router) Post(path string, handler http.Handler) {
	r.POST(path, wrapHandler(handler))
}

func (r *router) Put(path string, handler http.Handler) {
	r.PUT(path, wrapHandler(handler))
}

func (r *router) Delete(path string, handler http.Handler) {
	r.DELETE(path, wrapHandler(handler))
}

func NewRouter() *router {
	return &router{httprouter.New()}
}

func wrapHandler(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		context.Set(r, "params", ps)
		h.ServeHTTP(w, r)
	}
}

func getMongoSession() *mgo.Session {
	if mgoSession == nil {
		var err error

		mgoSession, err = mgo.Dial("127.0.0.1:27017")

		if err != nil {
			panic(err)
		}

		defer mgoSession.Close()
	}

	return mgoSession.Copy()
}

func main() {
	session := getMongoSession()
	appC := appContext{session.DB(dbName)}

	//// just testing begin

	// pageRepo := Repository{appC.db.C("pages")}
	// page := &Page{}
	// err := pageRepo.coll.Find(bson.M{"url": "lymphomahub.com"}).One(page)
	// if err != nil {
	// 	log.Printf("Page '%s' doesn't exist", err.Error())
	// 	panic(err)
	// }

	// pageEntry := &PageEntry{
	// 	Load: time.Now().Second(),
	// 	Page: page.Id,
	// }
	// pageEntry.SetInsertDefaults(time.Now())

	// entriesRepo := Repository{appC.db.C("page_entry")}
	// err = entriesRepo.coll.Insert(pageEntry)
	// if err != nil {
	// 	log.Printf("Load entry couldn't be created!! '%s'", err.Error())
	// }

	//// just testing end

	commonHandlers := alice.New(context.ClearHandler, loggingHandler, recoverHandler, acceptHandler)
	router := NewRouter()
	router.Get("/page/:id", commonHandlers.ThenFunc(appC.pageHandler))
	router.Put("/page/:id", commonHandlers.Append(contentTypeHandler, bodyHandler(PageResource{})).ThenFunc(appC.updatepageHandler))
	router.Delete("/page/:id", commonHandlers.ThenFunc(appC.deletepageHandler))
	router.Get("/pages", commonHandlers.ThenFunc(appC.pagesHandler))
	router.Post("/page", commonHandlers.Append(contentTypeHandler, bodyHandler(PageResource{})).ThenFunc(appC.createpageHandler))
	http.ListenAndServe(":8080", router)
}
