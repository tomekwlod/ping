package db

import (
	"fmt"
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
	fmt.Println("ENV:", configPath)
	if configPath == "" {
		// or Panic and env should be everytime present, even on dev
		configPath = "../../configs"
	}

	cnf := ping.DBConfig{}
	if err := configor.Load(&cnf, configPath+"/db.yml"); err != nil {
		panic(err)
	}

	if session == nil {
		var err error
		log.Println("Connecting to ", cnf.Addr+":"+cnf.Port)
		session, err = mgo.Dial(cnf.Addr + ":" + cnf.Port)

		if err != nil {
			panic(err)
		}

		session.SetMode(mgo.Monotonic, true)

		if err = session.DB(cnf.Database).C("page").EnsureIndex(mgo.Index{
			Key:    []string{"url"},
			Unique: true,
		}); err != nil {
			log.Print("context: ", err)
		}

		defer session.Close()
	}

	return session.Copy()
}
