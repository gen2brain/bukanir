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

// eztv type
type eztv struct {
	Host  string
	Query string
}

// NewEztv returns new eztv
func NewEztv(host string) *eztv {
	return &eztv{host, ""}
}

// Search returns torrents for query
func (t *eztv) Search(query string) ([]TTorrent, error) {
	var results []TTorrent
	uri := fmt.Sprintf("http://%s/search/%s", t.Host, url.QueryEscape(query))

	t.Query = query

	if verbose {
		log.Printf("EZTV: GET %s\n", uri)
	}

	doc, err := getDocument(uri)
	if err != nil {
		return nil, err
	}

	if doc != nil {
		results, err = t.getTorrents(doc)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// getTorrents returns torrents from html page
func (t *eztv) getTorrents(doc *goquery.Document) ([]TTorrent, error) {
	var results []TTorrent
	trs := doc.Find(`tr.forum_header_border`)

	if trs.Length() == 0 {
		return nil, errors.New(fmt.Sprintf("No results"))
	}

	var w sync.WaitGroup
	parseHTML := func(i int, s *goquery.Selection) {
		defer w.Done()

		td1 := s.Find(`td`).First().Next()
		td2 := s.Find(`td`).First().Next().Next()
		td3 := s.Find(`td`).First().Next().Next().Next()
		td4 := s.Find(`td`).First().Next().Next().Next().Next().Next()

		title := td1.Find(`a`).First().Text()
		magnet, _ := td2.Find(`a`).Attr(`href`)
		seeders, _ := strconv.Atoi(strings.Replace(td4.Text(), ",", "", -1))

		formattedTitle := getTitle(title)

		if seeders == 0 || formattedTitle == "" || !strings.HasPrefix(magnet, "magnet:?") {
			return
		}

		if !strings.Contains(strings.ToLower(formattedTitle), strings.ToLower(t.Query)) {
			return
		}

		var size uint64
		var sizeHuman string
		size, _ = humanize.ParseBytes(td3.Text())
		sizeHuman = humanize.IBytes(size)

		if size == 0 || size > 5120*1024*1024 {
			return
		}

		season, _ := strconv.Atoi(getSeason(title))
		episode, _ := strconv.Atoi(getEpisode(title))

		if season == 0 && episode == 0 {
			return
		}

		t := TTorrent{
			title,
			formattedTitle,
			magnet,
			getYear(title),
			int64(size),
			sizeHuman,
			seeders,
			CategoryTV,
			season,
			episode,
		}

		results = append(results, t)
	}

	w.Add(trs.Length())
	trs.Each(func(i int, s *goquery.Selection) {
		go parseHTML(i, s)
	})
	w.Wait()

	return results, nil
}
