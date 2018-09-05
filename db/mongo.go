package db

import (
	"log"
	"sync"
	"time"

	"github.com/tomekwlod/ping"
	mgo "gopkg.in/mgo.v2"
)

// CreateSession creates a new master mongodb session which you should next Clone or Copy
func CreateSession(host string) (session *mgo.Session, err error) {
	mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    []string{host},
		Timeout:  60 * time.Second,
		Database: "ping",
		// Username: AuthUserName,
		// Password: AuthPassword,
	}

	// Create a session which maintains a pool of socket connections to our MongoDB.
	session, err = mgo.DialWithInfo(mongoDBDialInfo)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// func NewMongoSession() *mgo.Session {
// 	mongoDBDialInfo := &mgo.DialInfo{
// 		Addrs:    []string{"127.0.0.1:27021"},
// 		Timeout:  60 * time.Second,
// 		Database: "ping",
// 		// Username: AuthUserName,
// 		// Password: AuthPassword,
// 	}

// 	// Create a session which maintains a pool of socket connections
// 	// to our MongoDB.
// 	mongoSession, err := mgo.DialWithInfo(mongoDBDialInfo)
// 	if err != nil {
// 		log.Fatalf("CreateSession: %s\n", err)
// 	}

// 	// Reads may not be entirely up-to-date, but they will always see the
// 	// history of changes moving forward, the data read will be consistent
// 	// across sequential queries in the same session, and modifications made
// 	// within the session will be observed in following queries (read-your-writes).
// 	// http://godoc.org/labix.org/v2/mgo#Session.SetMode
// 	mongoSession.SetMode(mgo.Monotonic, true)

// 	if err = mongoSession.DB("ping").C("page").EnsureIndex(mgo.Index{
// 		Key:    []string{"url"},
// 		Unique: true,
// 	}); err != nil {
// 		log.Print("context: ", err)
// 	}

// 	return mongoSession
// }

// func MongoSession() *mgo.Session {
// 	cnf := ping.LoadConfig()

// 	if session == nil {
// 		var err error

// 		mongoDBDialInfo := &mgo.DialInfo{
// 			Addrs:    []string{cnf.MongoDB_Addr + ":" + cnf.MongoDB_Port},
// 			Timeout:  60 * time.Second,
// 			Database: cnf.MongoDB_Database,
// 			// Username: AuthUserName,
// 			// Password: AuthPassword,
// 		}

// 		log.Println("Connecting to ", cnf.MongoDB_Addr+":"+cnf.MongoDB_Port)
// 		session, err = mgo.DialWithInfo(mongoDBDialInfo)
// 		defer session.Close()

// 		if err != nil {
// 			panic(err)
// 		}

// 		session.SetMode(mgo.Monotonic, true)

// 		if err = session.DB(cnf.MongoDB_Database).C("page").EnsureIndex(mgo.Index{
// 			Key:    []string{"url"},
// 			Unique: true,
// 		}); err != nil {
// 			log.Print("context: ", err)
// 		}
// 	}

// 	return session.Copy()
// }

// run it like this:
// // Create a wait group to manage the goroutines.
// var waitGroup sync.WaitGroup

// // Perform 10 concurrent queries against the database.
// waitGroup.Add(10)
// for query := 0; query < 10; query++ {
// 	go db.RunQueryXTimes(query, &waitGroup, db.NewMongoSession())
// }

// // Wait for all the queries to complete.
// waitGroup.Wait()
// log.Println("All Queries Completed")
func RunQueryXTimes(numberOfQueries int, waitGroup *sync.WaitGroup, mongoSession *mgo.Session) {
	// Decrement the wait group count so the program knows this
	// has been completed once the goroutine exits.
	defer waitGroup.Done()

	// Request a socket connection from the session to process our query.
	// Close the session when the goroutine exits and put the connection back
	// into the pool.
	sessionCopy := mongoSession.Copy()
	defer sessionCopy.Close()

	// Get a collection to execute the query against.
	collection := sessionCopy.DB("ping").C("page")

	log.Printf("RunQuery : %d : Executing\n", numberOfQueries)

	// Retrieve the list of stations.
	var page []ping.Page
	err := collection.Find(nil).All(&page)
	if err != nil {
		log.Printf("RunQuery : ERROR : %s\n", err)
		return
	}

	log.Printf("RunQuery : %d : Count[%d]\n", numberOfQueries, len(page))
}
