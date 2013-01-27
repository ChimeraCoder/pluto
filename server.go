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
	"time"
)

const BCRYPT_COST = 12
const RSS_TIMEOUT = 100

var (
	httpAddr        = flag.String("addr", ":8000", "HTTP server address")
	baseTmpl string = "templates/base.tmpl"
	store           = sessions.NewCookieStore([]byte(COOKIE_SECRET))

	//The following three variables can be defined using environment variables
	//to avoid committing them by mistake
	//Alternatively, place variable declarations in a separate conf.go file
	//which is already in the .gitignore file

	//COOKIE_SECRET = []byte(os.Getenv("COOKIE_SECRET"))
	//APP_ID = os.Getenv("APP_ID")
	//APP_SECRET = os.Getenv("APP_SECRET")
)

//Create an array of URLs that point to the RSS feeds for each of the fellows' blogs
var FELLOW_BLOGS = []string{"http://blog.goneill.net/rss"}

func scrapeRss(uri string) {
	uri = "http://blog.goneill.net/rss"
	feed := rss.New(RSS_TIMEOUT, true, chanHandler, itemHandler)
	for {
		if err := feed.Fetch(uri, nil); err != nil {
			fmt.Fprintf(os.Stderr, "[e.fetch] %s: %s", uri, err)
			return
		}

		log.Printf("Sleeping for %d seconds on %s", feed.SecondsTillUpdate(), uri)
		<-time.After(time.Duration(feed.SecondsTillUpdate() * 1e9))
	}
}

func chanHandler(feed *rss.Feed, newchannels []*rss.Channel) {
	log.Printf("Found %d new channel(s) in %s", len(newchannels), feed.Url)
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {
	log.Printf("Found %d new item(s) in %s", len(newitems), feed.Url)
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

	//Create a unique index on 'guid', so that entires will not be duplicated
	//Any duplicate entries will be dropped silently when insertion is attempted
	guid_index := mgo.Index{
		Key:        []string{"guid"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	mongodb_session.DB(MONGODB_DATABASE).C("blogposts").EnsureIndex(guid_index)

	//Set off a separate goroutine for each fellow's blog to keep it continuously up-to-date
	for _, rss_uri := range FELLOW_BLOGS {
		go scrapeRss(rss_uri)
	}

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
