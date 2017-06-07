package main

import (
	"log"

	"github.com/gen2brain/bukanir/lib"
)

// Client type
type Client struct {
	CacheDir string
}

// NewClient returns new Client
func NewClient() *Client {
	return &Client{cacheDir()}
}

// Top movies
func (c *Client) Top(widget *List, category, limit, force, cacheDays int, tpbHost string) {
	data, err := bukanir.Category(category, limit, force, c.CacheDir, int64(cacheDays), tpbHost)
	if err != nil {
		log.Printf("ERROR: Category: %s\n", err.Error())
		widget.Finished("")
		return
	}

	widget.Finished(data)
}

// Search movies
func (c *Client) Search(widget *List, query string, limit, force, cacheDays int, pages int, tpbHost string, eztvHost string) {
	data, err := bukanir.Search(query, limit, force, c.CacheDir, int64(cacheDays), pages, tpbHost, eztvHost)
	if err != nil {
		log.Printf("ERROR: Search: %s\n", err.Error())
		widget.Finished("")
		return
	}

	widget.Finished(data)
	return
}

// Summary for movie
func (c *Client) Summary(widget *Summary, m bukanir.TMovie) {
	data, err := bukanir.Summary(m.Id, m.Category, m.Season, m.Episode)
	if err != nil {
		log.Printf("ERROR: Summary: %s\n", err.Error())
		widget.Finished("")
		return
	}

	widget.Finished(data)
}

// Complete search query
func (c *Client) Complete(widget *Toolbar, text string) {
	data, err := bukanir.AutoComplete(text, 10)
	if err != nil {
		log.Printf("ERROR: AutoComplete: %s\n", err.Error())
		widget.Finished("")
		return
	}

	widget.Finished(data)
}

// Popular movies
func (c *Client) Popular(widget *Toolbar) {
	data, err := bukanir.Popular()
	if err != nil {
		log.Printf("ERROR: Popular: %s\n", err.Error())
		widget.Finished2("")
		return
	}

	widget.Finished2(data)
}

// TopRated movies
func (c *Client) TopRated(widget *Toolbar) {
	data, err := bukanir.TopRated()
	if err != nil {
		log.Printf("ERROR: TopRated: %s\n", err.Error())
		widget.Finished3("")
		return
	}

	widget.Finished3(data)
}

// Genres list
func (c *Client) Genres(widget *Toolbar) {
	data, err := bukanir.Genres()
	if err != nil {
		log.Printf("ERROR: Genres: %s\n", err.Error())
		widget.Finished4("")
		return
	}

	widget.Finished4(data)
}

// Genre movies
func (c *Client) Genre(widget *List, id int, limit int, force int, cacheDays int, tpbHost string) {
	data, err := bukanir.Genre(id, limit, force, c.CacheDir, int64(cacheDays), tpbHost)
	if err != nil {
		log.Printf("ERROR: Genre: %s\n", err.Error())
		widget.Finished("")
		return
	}

	widget.Finished(data)
}
