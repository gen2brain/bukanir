package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/gen2brain/bukanir/lib/bukanir"
)

type Client struct {
	Mutex    sync.RWMutex
	CacheDir string
}

func NewClient() *Client {
	var mutex sync.RWMutex
	cacheDir := filepath.Join(homeDir(), ".cache", "bukanir")
	return &Client{mutex, cacheDir}
}

func (c *Client) Top(widget *List, category, limit, force, cacheDays int, host string) {
	data, err := bukanir.Category(category, limit, force, c.CacheDir, int64(cacheDays), host)
	if err != nil {
		log.Printf("ERROR: Category: %s\n", err.Error())
		widget.Finished("")
		return
	}

	widget.Finished(data)
}

func (c *Client) Search(widget *List, query string, limit, force, cacheDays int, pages int, host string) {
	data, err := bukanir.Search(query, limit, force, c.CacheDir, int64(cacheDays), pages, host)
	if err != nil {
		log.Printf("ERROR: Search: %s\n", err.Error())
		widget.Finished("")
		return
	}

	widget.Finished(data)
	return
}

func (c *Client) Summary(widget *Summary, m bukanir.TMovie) {
	data, err := bukanir.Summary(m.Id, m.Category, m.Season, m.Episode)
	if err != nil {
		log.Printf("ERROR: Summary: %s\n", err.Error())
		widget.Finished("")
		return
	}

	res, err := http.Get(m.PosterXLarge)
	if err != nil {
		log.Printf("ERROR: Get: %s\n", err.Error())
		widget.Finished("")
		return
	}

	poster, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("ERROR: ReadAll: %s\n", err.Error())
		widget.Finished("")
		return
	}
	res.Body.Close()

	c.Mutex.Lock()
	posters[m.MagnetLink] = poster
	c.Mutex.Unlock()

	widget.Finished(data)
	widget.Finished2(m.MagnetLink)
}

func (c *Client) Complete(widget *Toolbar, text string) {
	data, err := bukanir.AutoComplete(text, 10)
	if err != nil {
		log.Printf("ERROR: AutoComplete: %s\n", err.Error())
		widget.Finished("")
		return
	}

	widget.Finished(data)
}

func (c *Client) Popular(widget *Toolbar) {
	data, err := bukanir.Popular()
	if err != nil {
		log.Printf("ERROR: Popular: %s\n", err.Error())
		widget.Finished2("")
		return
	}

	widget.Finished2(data)
}

func (c *Client) TopRated(widget *Toolbar) {
	data, err := bukanir.TopRated()
	if err != nil {
		log.Printf("ERROR: TopRated: %s\n", err.Error())
		widget.Finished3("")
		return
	}

	widget.Finished3(data)
}

func (c *Client) Genres(widget *Toolbar) {
	data, err := bukanir.Genres()
	if err != nil {
		log.Printf("ERROR: Genres: %s\n", err.Error())
		widget.Finished4("")
		return
	}

	widget.Finished4(data)
}

func (c *Client) Genre(widget *List, id int, limit int, force int, cacheDays int, host string) {
	data, err := bukanir.Genre(id, limit, force, c.CacheDir, int64(cacheDays), host)
	if err != nil {
		log.Printf("ERROR: Genre: %s\n", err.Error())
		widget.Finished("")
		return
	}

	widget.Finished(data)
}
