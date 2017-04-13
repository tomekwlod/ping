package utils

import mgo "gopkg.in/mgo.v2"

var (
	MgoSession *mgo.Session
	DbName     = "ping"
)

func GetMongoSession() *mgo.Session {
	if MgoSession == nil {
		var err error

		MgoSession, err = mgo.Dial("127.0.0.1:27017")

		if err != nil {
			panic(err)
		}

		MgoSession.SetMode(mgo.Monotonic, true)

		defer MgoSession.Close()
	}

	return MgoSession.Copy()
}
