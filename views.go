package main

import (
	"fmt"
	"net/http"
	"text/template"
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
