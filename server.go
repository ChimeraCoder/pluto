package main

import (
	rss "./go-pkg-rss"
	"flag"
	"fmt"
	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"labix.org/v2/mgo"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"
)

const BCRYPT_COST = 12

var (
	httpAddr               = flag.String("addr", ":8000", "HTTP server address")
	baseTmpl        string = "templates/base.tmpl"
	store                  = sessions.NewCookieStore([]byte(COOKIE_SECRET))

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
	s1, err := template.ParseFiles("templates/base.tmpl", "templates/index.tmpl")
	if err != nil {
		panic(err)
	}
	s1.ExecuteTemplate(w, "base", nil)
}


func scrapeRss(url string) {
	timeout := 100
	uri := "http://blog.goneill.net/rss"
	feed := rss.New(timeout, true, chanHandler, itemHandler)
	for {
		if err := feed.Fetch(uri, nil); err != nil {
			fmt.Fprintf(os.Stderr, "[e.fetch] %s: %s", uri, err)
			return
		}

		<-time.After(time.Duration(feed.SecondsTillUpdate() * 1e9))
	}
}

func chanHandler(feed *rss.Feed, newchannels []*rss.Channel) {
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {
	for _, item := range newitems {
		savePost(*item)
	}
}

//Given an RSS item, save it in mongodb
func savePost(post rss.Item) error {
	return withCollection("blogposts", func(c *mgo.Collection) error {
		return c.Insert(post)
	})
}

func main() {

	var err error

	log.Print("Dialing mongodb database")
	mongodb_session, err = mgo.Dial(MONGODB_URL)
	if err != nil {
		panic(err)
	}
	log.Print("Succesfully dialed mongodb database")

	r := pat.New()

	err = mongodb_session.DB(MONGODB_DATABASE).Login(MONGODB_USERNAME, MONGODB_PASSWORD)

	guid_index := mgo.Index{
		Key:        []string{"guid"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	mongodb_session.DB(MONGODB_DATABASE).C("blogposts").EnsureIndex(guid_index)

	//Order of routes matters
	//Routes *will* match prefixes 

	http.Handle("/static/", http.FileServer(http.Dir("public")))
	r.Get("/profile", serveProfile)
	r.Get("/", serveHome)
	http.Handle("/", r)

	if err := http.ListenAndServe(*httpAddr, nil); err != nil {
		log.Fatalf("Error listening, %v", err)
	}
}
