package main

import (
	"flag"
	"fmt"
    "github.com/gorilla/sessions"
	"log"
	"net/http"
	"text/template"
)

const BCRYPT_COST = 12

var (
	httpAddr        = flag.String("addr", ":8000", "HTTP server address")
	baseTmpl string = "templates/base.tmpl"
    store = sessions.NewCookieStore([]byte(COOKIE_SECRET))

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

func main() {
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/profile", serveProfile)
	http.Handle("/static/", http.FileServer(http.Dir("public")))

	if err := http.ListenAndServe(*httpAddr, nil); err != nil {
		log.Fatalf("Error listening, %v", err)
	}
}
