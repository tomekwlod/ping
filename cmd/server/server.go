package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/tomekwlod/ping"
	"github.com/tomekwlod/ping/db"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	ErrBadRequest           = &Error{"bad_request", 400, "Bad request", "Request body is not well-formed. It must be JSON."}
	ErrNotAcceptable        = &Error{"not_acceptable", 406, "Not Acceptable", "Accept header must be set to 'application/json'."}
	ErrUnsupportedMediaType = &Error{"unsupported_media_type", 415, "Unsupported Media Type", "Content-Type header must be set to: 'application/json'."}
	ErrInternalServer       = &Error{"internal_server_error", 500, "Internal Server Error", "Something went wrong."}
)

type appContext struct {
	db *mgo.Database
}

type repository struct {
	coll *mgo.Collection
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(Errors{[]*Error{err}})
}

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

// Here is my request and I would like (to Accept) this response format
// I expect to receive this format only
func acceptHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			WriteError(w, ErrNotAcceptable)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// Content-Type header tells the server what the attached data actually is
// Only for PUT & POST
func contentTypeHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
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

func (c *appContext) pagesHandler(w http.ResponseWriter, r *http.Request) {
	repo := repository{c.db.C("pages")}
	pages, err := repo.AllPages()
	if err != nil {
		panic(err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(pages)
}

func (c *appContext) pageHistoryHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)

	repo := repository{c.db.C("page_entry")}
	history, err := repo.AllPageHistory(params.ByName("id"))

	if err != nil {
		panic(err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(history)
}

func (c *appContext) pageHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	repo := repository{c.db.C("pages")}
	page, err := repo.Find(params.ByName("id"))
	if err != nil {
		panic(err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(page)
}

func (c *appContext) createpageHandler(w http.ResponseWriter, r *http.Request) {
	repo := repository{c.db.C("pages")}

	body := context.Get(r, "body").(*ping.SinglePage)
	body.Data.SetInsertDefaults(time.Now())

	err := repo.Create(&body.Data)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(body)
}

// allow CORS
func (c *appContext) allowCorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")

		w.WriteHeader(200)
	}
}

func (c *appContext) updatepageHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	repo := repository{c.db.C("pages")}

	body := context.Get(r, "body").(*ping.SinglePage)

	update := ping.SinglePage{}
	update.Data.Interval = body.Data.Interval
	update.Data.Description = body.Data.Description
	update.Data.Name = body.Data.Name
	update.Data.Url = body.Data.Url
	update.Data.RescueUrl = body.Data.RescueUrl

	update.Data.Id = bson.ObjectIdHex(params.ByName("id"))
	update.Data.SetUpdateDefaults(time.Now())

	err := repo.Update(&update.Data)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")

	w.WriteHeader(204)
	w.Write([]byte("\n"))
}

func (c *appContext) deletepageHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	repo := repository{c.db.C("pages")}
	err := repo.Delete(params.ByName("id"))
	if err != nil {
		panic(err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")

	w.WriteHeader(204)
	w.Write([]byte("\n"))
}

// Repos

func (r *repository) AllPages() (ping.PageCollection, error) {
	result := ping.PageCollection{[]ping.Page{}}
	err := r.coll.Find(bson.M{}).Select(bson.M{"content": 0}).All(&result.Data)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *repository) AllPageHistory(id string) (ping.PageEntryCollection, error) {
	result := ping.PageEntryCollection{[]ping.PageEntry{}}
	err := r.coll.Find(bson.M{"page": bson.ObjectIdHex(id)}).All(&result.Data)
	fmt.Println("Build the pagination")
	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *repository) Find(id string) (ping.SinglePage, error) {
	result := ping.SinglePage{}
	err := r.coll.FindId(bson.ObjectIdHex(id)).One(&result.Data)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *repository) Create(page *ping.Page) error {
	result := ping.SinglePage{}
	_ = r.coll.Find(bson.M{"url": page.Url}).One(&result.Data)

	if result.Data.Id != "" {
		panic("Page already exists")
	}

	id := bson.NewObjectId()

	if page.Url == "" {
		panic("Page cannot be created without the URL value")
	}

	_, err := r.coll.UpsertId(id, page)
	if err != nil {
		panic(err)
	}

	page.Id = id

	return nil
}

func (r *repository) Update(page *ping.Page) error {
	err := r.coll.UpdateId(page.Id, page)
	if err != nil {
		panic(err)
	}

	return nil
}

func (r *repository) Delete(id string) error {
	err := r.coll.RemoveId(bson.ObjectIdHex(id))
	if err != nil {
		panic(err)
	}

	return nil
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

func (r *router) Options(path string, handler http.Handler) {
	r.OPTIONS(path, wrapHandler(handler))
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

func port() string {
	port := os.Getenv("PING_PORT")

	if len(port) == 0 {
		port = "8080"
	}

	return port
}

func main() {
	cnf := ping.LoadConfig()
	mongoSession := db.MongoSession()
	appC := appContext{mongoSession.DB(cnf.MongoDB_Database)}

	commonHandlers := alice.New(context.ClearHandler, loggingHandler, recoverHandler, acceptHandler)
	optionsHandlers := alice.New(context.ClearHandler, loggingHandler)
	router := NewRouter()

	router.Get("/page/:id/history", commonHandlers.ThenFunc(appC.pageHistoryHandler))
	router.Get("/page/:id", commonHandlers.ThenFunc(appC.pageHandler))
	router.Put("/page/:id", commonHandlers.Append(contentTypeHandler, bodyHandler(ping.SinglePage{})).ThenFunc(appC.updatepageHandler))
	router.Delete("/page/:id", commonHandlers.ThenFunc(appC.deletepageHandler))
	router.Get("/pages", commonHandlers.ThenFunc(appC.pagesHandler))
	router.Post("/page", commonHandlers.Append(contentTypeHandler, bodyHandler(ping.SinglePage{})).ThenFunc(appC.createpageHandler))
	router.Options("/*name", optionsHandlers.ThenFunc(appC.allowCorsHandler))

	// curl -X POST -H 'Accept: application/json' -H 'Content-Type: application/json' -d '{"data": {"url":"http://website.com/api", "status":0, "interval":1}}' localhost:8080/page
	log.Printf("Server started and listening on port %s \n\n", port())
	if err := http.ListenAndServe(":"+port(), router); err != nil {
		log.Panic("Error occured: ", err)
	}
}
