package main

import (
	rss "./go-pkg-rss"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"html"
	"html/template"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const BCRYPT_COST = 12
const RSS_TIMEOUT = 100
const FEEDS_LIST_FILENAME = "feeds_list.txt"

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

func scrapeRss(uri string) {
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

func servePosts(w http.ResponseWriter, r *http.Request) {
	//TODO make this a proper query for the feeds we want
	var posts []rss.Item
	if err := withCollection("blogposts", func(c *mgo.Collection) error {
		return c.Find(bson.M{}).All(&posts)
	}); err != nil {
		panic(err)
	}

	log.Printf("Fetching %d posts", len(posts))

	//You may want to refactor this, as in renderTemplate, but this is how template inheritance works in Go
	funcs := template.FuncMap{
		"foo":            func(foo string) string { return foo },
		"UnescapeString": html.UnescapeString}
	s1, _ := template.ParseFiles("templates/base.tmpl", "templates/posts.tmpl")
	s1 = s1.Funcs(funcs)

	s1.ExecuteTemplate(w, "base", posts)

}

//Return a the feeds ([]rss.Item) serialized in JSON
func serveFeeds(w http.ResponseWriter, r *http.Request) {

	//TODO make this a proper query for the feeds we want
	var feeds []rss.Item
	if err := withCollection("blogposts", func(c *mgo.Collection) error {
		return c.Find(bson.M{}).All(&feeds)
	}); err != nil {
		panic(err)
	}
	bts, err := json.Marshal(feeds)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(bts))
	return
}

func main() {

	var err error

	log.Print("Dialing mongodb database")
	mongodb_session, err = mgo.Dial(MONGODB_URL)
	if err != nil {
		panic(err)
	}
	log.Print("Succesfully dialed mongodb database")

	err = mongodb_session.DB(MONGODB_DATABASE).Login(MONGODB_USERNAME, MONGODB_PASSWORD)

	r := pat.New()
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

	//Read a list of all fellows' rss feeds from a file
	bts, err := ioutil.ReadFile(FEEDS_LIST_FILENAME)
	if err != nil {
		panic(err)
	}

	feed_urls := strings.Split(strings.Trim(string(bts), "\n\r"), "\n")

	//Set off a separate goroutine for each fellow's blog to keep it continuously up-to-date
	for _, feed_url := range feed_urls {
		log.Printf("Found %s", feed_url)
		go func(uri string) {
			scrapeRss(uri)
		}(feed_url)
	}

	//Order of routes matters
	//Routes *will* match prefixes 
	http.Handle("/static/", http.FileServer(http.Dir("public")))
	r.Get("/feeds/all", serveFeeds)
	r.Get("/feeds/posts", servePosts)
	r.Get("/", serveHome)
	http.Handle("/", r)

	if err := http.ListenAndServe(*httpAddr, nil); err != nil {
		log.Fatalf("Error listening, %v", err)
	}
}
