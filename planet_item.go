package main

import (
	rss "github.com/jteeuwen/go-pkg-rss"
	"time"
)

type Item struct {
	// RSS and Shared fields
	Title         string
	Links         []*rss.Link
	Description   string
	Author        rss.Author
	Categories    []*rss.Category
	Comments      string
	Enclosures    []*rss.Enclosure
	Guid          string
	PubDateParsed *time.Time
	Source        *rss.Source

	// Atom specific fields
	Id           string
	Generator    *rss.Generator
	Contributors []string
	Content      *rss.Content
}

func NewItem(old rss.Item) (it Item, err error) {
	//Try to parse the date of publication, so we don't have to store it as a basic string
	//TODO this is super-hacky... yuck
	dt, err := time.Parse("Mon Jan 2 2006 15:04:05 GMT-0700 (MST)", old.PubDate)
	if err != nil {
		dt, err = time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", old.PubDate)
		if err != nil {
			dt, err = time.Parse("2006-01-02T15:04:05-07:00", old.PubDate)
			if err != nil {
				dt, err = time.Parse("Mon, 02 Jan 2006 15:04:05 GMT", old.PubDate)
				if err != nil {
					return
				}
			}
		}
	}
	it = Item{old.Title, old.Links, old.Description, old.Author, old.Categories, old.Comments, old.Enclosures, old.Guid, &dt, old.Source, old.Id, old.Generator, old.Contributors, old.Content}
	return
}
