package bukanir

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/dustin/go-humanize"
)

// tpb type
type tpb struct {
	Host string
}

// NewTpb returns new tpb
func NewTpb(host string) *tpb {
	return &tpb{Host: host}
}

// Top returns top torrents for category
func (t *tpb) Top(category int) ([]TTorrent, error) {
	var results []TTorrent
	uri := fmt.Sprintf("http://%s/top/%d", t.Host, category)

	if verbose {
		log.Printf("TPB: GET %s\n", uri)
	}

	doc, err := getDocument(uri)
	if err != nil {
		return nil, err
	}

	if doc != nil {
		results, err = t.getTorrents(doc, -1)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// Search returns torrents for query
func (t *tpb) Search(query string, page int, cats string) ([]TTorrent, error) {
	var results []TTorrent
	uri := fmt.Sprintf("http://%s/search/%s/%d/7/%s", t.Host, url.QueryEscape(query), page, cats)

	if verbose {
		log.Printf("TPB: GET %s\n", uri)
	}

	doc, err := getDocument(uri)
	if err != nil {
		return nil, err
	}

	if doc != nil {
		results, err = t.getTorrents(doc, page)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// getTorrents returns torrents from html page
func (t *tpb) getTorrents(doc *goquery.Document, page int) ([]TTorrent, error) {
	var results []TTorrent
	divs := doc.Find(`div.detName`)

	if divs.Length() == 0 {
		return nil, errors.New(fmt.Sprintf("No results on page %d", page))
	}

	var w sync.WaitGroup

	parseHTML := func(i int, s *goquery.Selection) {
		defer w.Done()

		parent := s.Parent()
		prev := parent.Prev().First()

		title := s.Find(`a.detLink`).Text()
		magnet, _ := parent.Find(`a[title="Download this torrent using magnet"]`).Attr(`href`)
		desc := parent.Find(`font.detDesc`).Text()
		seeders, _ := strconv.Atoi(parent.Next().Text())

		c, _ := prev.Find(`a[title="More from this category"]`).Last().Attr(`href`)
		category, _ := strconv.Atoi(strings.Replace(c, "/browse/", "", -1))

		if seeders == 0 || getTitle(title) == "" || !strings.HasPrefix(magnet, "magnet:?") {
			return
		}

		var size uint64
		var sizeHuman string
		parts := strings.Split(desc, ", ")
		if len(parts) > 1 {
			size, _ = humanize.ParseBytes(strings.Split(parts[1], " ")[1])
			sizeHuman = humanize.IBytes(size)
		}

		if size > 5120*1024*1024 {
			return
		}

		season, _ := strconv.Atoi(getSeason(title))
		episode, _ := strconv.Atoi(getEpisode(title))

		if category == CategoryTV || category == CategoryHDTV {
			if season == 0 && episode == 0 {
				return
			}
		}

		t := TTorrent{
			title,
			getTitle(title),
			magnet,
			getYear(title),
			int64(size),
			sizeHuman,
			seeders,
			category,
			season,
			episode,
		}

		results = append(results, t)
	}

	w.Add(divs.Length())
	divs.Each(func(i int, s *goquery.Selection) {
		go parseHTML(i, s)
	})
	w.Wait()

	return results, nil
}
