package main

import (
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"github.com/tomekwlod/ping"
	"gopkg.in/mgo.v2/bson"
)

var (
	errBadRequest           = &Error{"bad_request", 400, "Bad request", "Request body is not well-formed. It must be JSON."}
	errNotAcceptable        = &Error{"not_acceptable", 406, "Not Acceptable", "Accept header must be set to 'application/json'."}
	errUnsupportedMediaType = &Error{"unsupported_media_type", 415, "Unsupported Media Type", "Content-Type header must be set to: 'application/json'."}
	errInternalServer       = &Error{"internal_server_error", 500, "Internal Server Error", "Something went wrong."}
)

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

func wrapHandler(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		context.Set(r, "params", ps)
		h.ServeHTTP(w, r)
	}
}

// Middlewares
func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// s.logger.Printf("panic: %+v", err)
				WriteError(w, errInternalServer)
				return
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func (s *service) loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()

		s.logger.Printf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
		// log.Printf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

// Here is my request and I would like (to Accept) this response format
// I expect to receive this format only
func acceptHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			WriteError(w, errNotAcceptable)
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
			WriteError(w, errUnsupportedMediaType)
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
				WriteError(w, errBadRequest)
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

// allow CORS
func allowCorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")

		w.WriteHeader(200)
	}
}

// Main handlers
func (s *service) pagesHandler(w http.ResponseWriter, r *http.Request) {
	repo := s.getPageRepo()
	defer repo.Close()

	pages, err := repo.GetAll()
	if err != nil {
		WriteError(w, errBadRequest)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")
	w.Header().Set("Content-Type", "application/json")

	// should be json.NewEncoder(w).Encode(pages) where pages are PageCollection
	// below is a workaround to support data:{} responses
	type resp struct {
		Data []*ping.Page `json:"data"`
	}
	json.NewEncoder(w).Encode(resp{Data: pages})
}

func (s *service) pageHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)

	repo := s.getPageRepo()
	defer repo.Close()

	page, err := repo.Find(params.ByName("id"))
	if err != nil {
		s.logger.Panicln(err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(page)
}

func (s *service) createpageHandler(w http.ResponseWriter, r *http.Request) {
	body := context.Get(r, "body").(*ping.SinglePage)
	body.Data.SetInsertDefaults(time.Now())

	repo := s.getPageRepo()
	defer repo.Close()

	err := repo.Create(&body.Data)
	if err != nil {
		s.logger.Panicln(err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(body)
}

func (s *service) updatepageHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	body := context.Get(r, "body").(*ping.SinglePage)

	repo := s.getPageRepo()
	defer repo.Close()

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
		s.logger.Panicln(err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")

	w.WriteHeader(204)
	w.Write([]byte("\n"))
}

func (s *service) deletepageHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)

	repo := s.getPageRepo()
	defer repo.Close()

	err := repo.Delete(params.ByName("id"))
	if err != nil {
		s.logger.Panicln(err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, PUT")

	w.WriteHeader(204)
	w.Write([]byte("\n"))
}
