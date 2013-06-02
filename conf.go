package main

import (
	"io/ioutil"
	"launchpad.net/goyaml"
	"log"
)

var (
	auth             = false
	MONGODB_USERNAME = ""
	MONGODB_PASSWORD = ""
	MONGODB_DATABASE = "pluto"
	MONGODB_URL      = "localhost"
	COOKIE_SECRET    = ""
)

func parseConfigFile(path string) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var conf map[string]string
	err = goyaml.Unmarshal(file, &conf)
	if err != nil {
		log.Fatal(err)
	}
	if value, ok := conf["MONGODB_USERNAME"]; ok {
		if value, ok := conf["MONGODB_PASSWORD"]; ok {
			MONGODB_PASSWORD = value
		} else {
			log.Fatal("need username and password")
		}
		MONGODB_USERNAME = value
		auth = true
	}
	if value, ok := conf["MONGODB_PASSWORD"]; ok {
		if auth {
			MONGODB_PASSWORD = value
		} else {
			log.Fatal("need username and password")
		}
	}
	if value, ok := conf["MONGODB_DATABASE"]; ok {
		MONGODB_DATABASE = value
	}
	if value, ok := conf["MONGODB_URL"]; ok {
		MONGODB_URL = value
	}
	if value, ok := conf["COOKIE_SECRET"]; ok {
		COOKIE_SECRET = value
	}
}
