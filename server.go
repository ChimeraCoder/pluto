package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	rss "github.com/jteeuwen/go-pkg-rss"
	"html/template"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

const BCRYPT_COST = 12
const RSS_TIMEOUT = 100
const FEEDS_LIST_FILENAME = "feeds_list.txt"

const BLOGPOSTS_DB = "blogposts"

const POSTS_PER_PAGE = 10

var fetchposts = flag.Bool("fetchposts", false, "fetch blogposts and add them to the database")

var SANITIZE_REGEX = regexp.MustCompile(`<script.*?>.*?<\/script>`)
var AUTHOR_URL_REGEX = regexp.MustCompile(`(.*?)\/rss`)

var (
	httpAddr        = flag.String("addr", ":8000", "HTTP server address")
	baseTmpl string = "templates/base.tmpl"
	store           = sessions.NewCookieStore([]byte(COOKIE_SECRET))

	COOKIE_SECRET    = []byte(os.Getenv("COOKIE_SECRET"))
	MONGODB_URL      = os.Getenv("MONGODB_URL")
	MONGODB_USERNAME = os.Getenv("MONGODB_USERNAME")
	MONGODB_PASSWORD = os.Getenv("MONGODB_PASSWORD")
	MONGODB_DATABASE = os.Getenv("MONGODB_DATABASE")
)

func scrapeRss(uri string, author string) {
	feed := rss.New(RSS_TIMEOUT, true, chanHandler, customItemHandler(author))
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

func customItemHandler(author string) func(*rss.Feed, *rss.Channel, []*rss.Item) {
	return func(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {
		log.Printf("Found %d new item(s) in %s", len(newitems), feed.Url)
		for _, item := range newitems {
			log.Printf("Item is %+v", item)

			//If the author's name isn't defined, we should add it
			if item.Author.Name == "" {
				item.Author.Name = author
				item.Author.Uri = AUTHOR_URL_REGEX.FindStringSubmatch(feed.Url)[1]
			}

			item_updated, err := NewItem(*item)
			if err != nil {
				panic(err)
			}
			savePost(item_updated)
		}
	}
}

//Given an RSS item, save it in mongodb
func savePost(post Item) error {
	return withCollection(BLOGPOSTS_DB, func(c *mgo.Collection) error {
		return c.Insert(post)
	})
}

func servePosts(w http.ResponseWriter, r *http.Request) {

	var err error
	//Default to providing the first page
	page := 1
	page_s := r.FormValue("page")
	if page_s != "" {
		//No page specified
		page, err = strconv.Atoi(page_s)
		if err != nil {
			panic(err)
		}
	}

	//TODO make this a proper query for the feeds we want
	var posts []Item
	if err := withCollection(BLOGPOSTS_DB, func(c *mgo.Collection) error {
		return c.Find(bson.M{}).Skip(POSTS_PER_PAGE * (page - 1)).Limit(POSTS_PER_PAGE).Sort("-pubdateparsed").All(&posts)
	}); err != nil {
		panic(err)
	}

	log.Printf("Fetching %d posts", len(posts))

	renderHtml := func(raw_html string) template.HTML {
		return template.HTML(raw_html)
	}

	funcs := template.FuncMap{
		"RenderHtml": renderHtml,
	}

	authors, err := allAuthors()
	if err != nil {
		panic(err)
	}

	s1 := template.New("base").Funcs(funcs)
	s1, err = s1.ParseFiles("templates/base.tmpl", "templates/posts.tmpl")
	//s1, err := template.ParseFiles("templates/base.tmpl", "templates/posts.tmpl")
	if err != nil {
		panic(err)
	}
	s1 = s1.Funcs(funcs)

	s1.ExecuteTemplate(w, "base", struct {
		Posts   []Item
		Authors []rss.Author
	}{posts, authors})

}

//Return a the feeds ([]rss.Item) serialized in JSON
func serveFeeds(w http.ResponseWriter, r *http.Request) {

	var err error
	//Default to providing the first page
	page := 1
	page_s := r.FormValue("page")
	if page_s != "" {
		//No page specified
		page, err = strconv.Atoi(page_s)
		if err != nil {
			panic(err)
		}
	}

	log.Printf("Page %s", page_s)
	log.Print(page)

	//TODO make this a proper query for the feeds we want
	var posts []Item
	if err := withCollection(BLOGPOSTS_DB, func(c *mgo.Collection) error {
		return c.Find(bson.M{}).Skip(POSTS_PER_PAGE * (page - 1)).Limit(POSTS_PER_PAGE).All(&posts)
	}); err != nil {
		panic(err)
	}

	items_sanitized := make([]Item, len(posts))
	for i, item := range posts {
		items_sanitized[i] = sanitizeItem(item)
	}

	bts, err := json.Marshal(posts)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(bts))
	return
}

func allAuthors() (authors []rss.Author, err error) {
	err = withCollection(BLOGPOSTS_DB, func(c *mgo.Collection) error {
		return c.Find(bson.M{}).Distinct("author", &authors)
	})
	return
}

func serveAuthorInfo(w http.ResponseWriter, r *http.Request) {
	authors, err := allAuthors()
	if err != nil {
		panic(err)
	}

	bts, err := json.Marshal(authors)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(bts))
	return
}

//sanitizeItem sanitizes the HTML content by removing Javascript, etc.
//TODO make this not a terrible hack
func sanitizeItem(item Item) Item {
	//This is not currently safe to use for untrusted input, as it can be exploited trivially
	//However, it requires thought to exploit it, so it should prevent _accidental_ javascript spillage
	//It cannot remove javascript embedded in tag attributes (such as 'onclick:', etc.)
	item.Description = SANITIZE_REGEX.ReplaceAllString(item.Description, "")
	return item
}

func parseFeeds(filename string) ([][]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	return rows, err
}

func main() {
	flag.Parse()

	port_env := os.Getenv("PORT")
	if port_env != "" {
		*httpAddr = ":" + port_env
	}

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
	mongodb_session.DB(MONGODB_DATABASE).C(BLOGPOSTS_DB).EnsureIndex(guid_index)

	if *fetchposts {
		feeds, err := parseFeeds(FEEDS_LIST_FILENAME)
		if err != nil {
			panic(err)
		}

		for _, feed_info := range feeds {
			if len(feed_info) != 2 {
				panic(fmt.Errorf("Expect csv with 2 elements per row; received row with %d elements", len(feed_info)))
			}
			feed_url := feed_info[0]
			feed_author := feed_info[1]
			log.Printf("Found %s", feed_url)
			go func(uri string, author string) {
				scrapeRss(uri, author)
			}(feed_url, feed_author)
		}
	} else {
		log.Print("Skipping fetching posts - blog posts will NOT be updated")
	}

	//Order of routes matters
	//Routes *will* match prefixes 
	http.Handle("/static/", http.FileServer(http.Dir("public")))
	r.Get("/feeds/all", serveFeeds)
	r.Get("/authors/all", serveAuthorInfo)
	r.Get("/", servePosts)
	//r.Get("/", serveHome)
	http.Handle("/", r)

	if err := http.ListenAndServe(*httpAddr, nil); err != nil {
		log.Fatalf("Error listening, %v", err)
	}
}
