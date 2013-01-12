package main

import (
	"flag"
	"fmt"
	"time"
	"os"
	"github.com/gorilla/sessions"
	"labix.org/v2/mgo"
	"log"
	"net/http"
	"text/template"
	rss "github.com/jteeuwen/go-pkg-rss" 
)

const BCRYPT_COST = 12

var (
	httpAddr               = flag.String("addr", ":8000", "HTTP server address")
	baseTmpl        string = "templates/base.tmpl"
	store                  = sessions.NewCookieStore([]byte(COOKIE_SECRET))
	mongodb_session *mgo.Session

	//The following three variables can be defined using environment variables
	//to avoid committing them by mistake
	//Alternatively, place variable declarations in a separate conf.go file
	//which is already in the .gitignore file

	//COOKIE_SECRET = []byte(os.Getenv("COOKIE_SECRET"))
	//APP_ID = os.Getenv("APP_ID")
	//APP_SECRET = os.Getenv("APP_SECRET")
)

func serveProfile(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "This is where the user's profile information goes!")
	return
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	//The "/" path will be matched by default, so we need to check for a 404 error
	//you can use mux or something similar to refactor this part
	if r.URL.Path != "/" {
		http.Error(w, "Not Found", http.StatusNotFound)
	} else {
		//You may want to refactor this, but this is how template inheritance works in Go
		s1, _ := template.ParseFiles("templates/base.tmpl", "templates/index.tmpl")
		s1.ExecuteTemplate(w, "base", nil)
	}
}

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

func scrapeRss(url string) {
	timeout := 100
	feed := rss.New(timeout, true, chanHandler, itemHandler)
}

func chanHandler(feed *rss.Feed, newchannels []*rss.Channel) {
	uri := "http://blog.goneill.net/rss";
	for {
		if err := feed.Fetch(uri, nil); err != nil {
			fmt.Fprintf(os.Stderr, "[e.fetch] %s: %s", uri, err)
			return
		}

		<-time.After(time.Duration(feed.SecondsTillUpdate() * 1e9))
	}
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {
}

func main() {

	var err error
	mongodb_session, err = mgo.Dial(MONGODB_URL)
	if err != nil {
		panic(err)
	}

	err = mongodb_session.DB(MONGODB_DATABASE).Login(MONGODB_USERNAME, MONGODB_PASSWORD)

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/profile", serveProfile)
	http.Handle("/static/", http.FileServer(http.Dir("public")))

	if err := http.ListenAndServe(*httpAddr, nil); err != nil {
		log.Fatalf("Error listening, %v", err)
	}
}
