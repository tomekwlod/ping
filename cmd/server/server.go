package main

// Dependency Injection example : https://play.golang.org/p/u5XFzNAT-Ne

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/context"
	"github.com/justinas/alice"
	"github.com/tomekwlod/ping"
	"github.com/tomekwlod/ping/db"
	mgo "gopkg.in/mgo.v2"
)

func mgoHost() (host string) {
	// Database host from the environment variables
	host = os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost:27017"
	}

	return
}

func port() (port string) {
	port = os.Getenv("PING_PORT")
	if port == "" {
		port = "8080"
	}

	return
}

// service struct to hold the db and the logger
type service struct {
	session *mgo.Session
	logger  *log.Logger
}

// functions for the service struct
func (s *service) getPageRepo() ping.IPageRepository {
	return &ping.PageRepository{Session: s.session.Clone()}
}

func main() {
	// definig the logger & a log file
	file, err := os.OpenFile("log/http.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file", err)
	}
	multi := io.MultiWriter(file, os.Stdout)
	l := log.New(multi, "", log.Ldate|log.Ltime|log.Lshortfile)

	// definging the mongodb session
	mgoSession, err := db.CreateSession(mgoHost())
	defer mgoSession.Close()
	if err != nil {
		log.Panic("Cannot connect to Mongodb: ", err)
	}

	// combine the datastore session and the logger into one struct
	s := &service{
		session: mgoSession,
		logger:  l}

	/// testing here start
	//////////////////////
	// r := s.getPageRepo()
	// defer r.Close()

	// err = r.Create(&ping.Page{
	// 	Name:        "test5",
	// 	Url:         "http://www.amlglobalportal.com/api/v1/core/jr/status/content",
	// 	NextPing:    time.Now(),
	// 	Description: "test5",
	// 	Interval:    1,
	// 	LastStatus:  200,
	// 	Content:     "",
	// 	Disabled:    false,
	// })
	// if err != nil {
	// 	log.Panicln(err)
	// }
	// log.Println("ok")
	// return
	////////////////////
	/// testing here end

	commonHandlers := alice.New(context.ClearHandler, s.loggingHandler, recoverHandler, acceptHandler)
	optionsHandlers := alice.New(context.ClearHandler, s.loggingHandler)

	router := NewRouter()
	// router.Get("/page/:id/history", commonHandlers.ThenFunc(appC.pageHistoryHandler))
	router.Get("/page/:id", commonHandlers.ThenFunc(s.pageHandler))
	router.Put("/page/:id", commonHandlers.Append(contentTypeHandler, bodyHandler(ping.SinglePage{})).ThenFunc(s.updatepageHandler))
	router.Delete("/page/:id", commonHandlers.ThenFunc(s.deletepageHandler))
	router.Get("/pages", commonHandlers.ThenFunc(s.pagesHandler))
	router.Post("/page", commonHandlers.Append(contentTypeHandler, bodyHandler(ping.SinglePage{})).ThenFunc(s.createpageHandler))
	router.Options("/*name", optionsHandlers.ThenFunc(allowCorsHandler))

	// curl -X POST -H 'Accept: application/json' -H 'Content-Type: application/json' -d '{"data": {"url":"http://website.com/api", "status":0, "interval":1}}' localhost:8080/page
	l.Printf("Server started and listening on port %s. Ready for the requests.\n\n", port())
	if err := http.ListenAndServe(":"+port(), router); err != nil {
		l.Panic("Error occured: ", err)
	}
}
