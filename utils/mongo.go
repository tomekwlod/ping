package utils

import (
	"log"

	"github.com/tomekwlod/ping/config"

	mgo "gopkg.in/mgo.v2"
)

var (
	session  *mgo.Session
	address  = config.Params.DB.Address
	port     = config.Params.DB.Port
	database = config.Params.DB.DbName
)

func MongoSession() *mgo.Session {
	if session == nil {
		var err error

		session, err = mgo.Dial(address + ":" + port)

		if err != nil {
			panic(err)
		}

		session.SetMode(mgo.Monotonic, true)

		if err = session.DB(database).C("page").EnsureIndex(mgo.Index{
			Key:    []string{"url"},
			Unique: true,
		}); err != nil {
			log.Print("context: ", err)
		}

		defer session.Close()
	}

	return session.Copy()
}
