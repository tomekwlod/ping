package db

import (
	"log"
	"os"

	"github.com/jinzhu/configor"
	"github.com/tomekwlod/ping"
	mgo "gopkg.in/mgo.v2"
)

var (
	session *mgo.Session
)

func MongoSession() *mgo.Session {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		// or Panic and env should be everytime present, even on dev
		configPath = "../../configs"
	}

	cnf := ping.Parameters{}
	if err := configor.Load(&cnf, configPath+"/parameters.yml"); err != nil {
		panic(err)
	}

	if session == nil {
		var err error
		log.Println("Connecting to ", cnf.MongoDB_Addr+":"+cnf.MongoDB_Port)
		session, err = mgo.Dial(cnf.MongoDB_Addr + ":" + cnf.MongoDB_Port)

		if err != nil {
			panic(err)
		}

		session.SetMode(mgo.Monotonic, true)

		if err = session.DB(cnf.MongoDB_Database).C("page").EnsureIndex(mgo.Index{
			Key:    []string{"url"},
			Unique: true,
		}); err != nil {
			log.Print("context: ", err)
		}

		defer session.Close()
	}

	return session.Copy()
}
