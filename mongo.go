package main


import (

	"labix.org/v2/mgo"
)


var mongodb_session *mgo.Session

func mongodbSession() *mgo.Session {
	//Adapted from
	//http://denis.papathanasiou.org/?p=1090
	if mongodb_session == nil {
		var err error
		mongodb_session, err = mgo.Dial(MONGODB_URL)
		if err != nil {
			panic(err)
		}
		if err := mongodb_session.DB(MONGODB_DATABASE).Login(MONGODB_USERNAME, MONGODB_PASSWORD); err != nil {
			panic(err)
		}
	}
	return mongodb_session.Clone()
}

//Given the name of a mongodb collection and a function that runs on a mongodb collection, fetch a new mgo session and run the function on the collection with that name
func withCollection(collection_name string, f func(*mgo.Collection) error) error {
	mgo_session := mongodbSession()
	defer mgo_session.Close()
	coll := mgo_session.DB(MONGODB_DATABASE).C(collection_name)
	return f(coll)
}
