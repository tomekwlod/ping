package utils

import (
	"log"

	mgo "gopkg.in/mgo.v2"
)

var (
	mgoSession *mgo.Session
	dbName     = "ping"
	address    = "127.0.0.1"
	port       = "27017"
)

func GetMongoSession() *mgo.Session {
	if mgoSession == nil {
		var err error

		mgoSession, err = mgo.Dial(address + ":" + port)

		if err != nil {
			panic(err)
		}

		mgoSession.SetMode(mgo.Monotonic, true)

		if err = mgoSession.DB(dbName).C("page").EnsureIndex(mgo.Index{
			Key:    []string{"url"},
			Unique: true,
		}); err != nil {
			log.Print("context: ", err)
		}

		defer mgoSession.Close()
	}

	return mgoSession.Copy()
}
