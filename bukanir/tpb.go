package bukanir

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dustin/go-humanize"
)

var hosts = []string{
	"thepiratebay.org",
	"thepiratebay.mk",
	"thepiratebay.cd",
	"thepiratebay.lv",
}

type tpb struct {
	Host string
}

type torrent struct {
	Title          string
	FormattedTitle string
	MagnetLink     string
	Year           string
	Size           uint64
	SizeHuman      string
	Seeders        int
	Category       int
	Season         int
	Episode        int
}

var (
	category_movies   int = 201
	category_hdmovies int = 207
	category_tv       int = 205
	category_hdtv     int = 208
)

var categories = []int{
	category_movies,
	category_hdmovies,
	category_tv,
	category_hdtv,
}

func tpbInit(host string) *tpb {
	return &tpb{Host: host}
}

func (t *tpb) Top(category int) ([]torrent, error) {
	var results []torrent
	uri := "https://%s/top/%d"

	doc, err := getDocument(fmt.Sprintf(uri, t.Host, category))
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

func (t *tpb) Search(query string, page int) ([]torrent, error) {
	var results []torrent
	uri := "https://%s/search/%s/%d/7/201,207,205,208"

	doc, err := getDocument(fmt.Sprintf(uri, t.Host, url.QueryEscape(query), page))
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

func (t *tpb) getTorrents(doc *goquery.Document) ([]torrent, error) {
	var results []torrent
	divs := doc.Find(`div.detName`)

	if divs.Length() == 0 {
		return nil, errors.New("No divs with class detName")
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

		if seeders == 0 || getTitle(title) == "" {
			return
		}

		var size uint64
		var sizeHuman string
		parts := strings.Split(desc, ", ")
		if len(parts) > 1 {
			size, _ = humanize.ParseBytes(strings.Split(parts[1], " ")[1])
			sizeHuman = humanize.IBytes(size)
		}

		if size > 4095*1024*1024 {
			return
		}

		season, _ := strconv.Atoi(getSeason(title))
		episode, _ := strconv.Atoi(getEpisode(title))

		if category == category_tv || category == category_hdtv {
			if season == 0 && episode == 0 {
				return
			}
		}

		t := torrent{
			title,
			getTitle(title),
			magnet,
			getYear(title),
			size,
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

func getHost() string {
	for _, host := range hosts {
		_, err := net.DialTimeout("tcp", host+":80", time.Duration(5)*time.Second)
		if err == nil {
			if verbose {
				log.Printf("Using host %s\n", host)
			}
			return host
		}
	}
	if verbose {
		log.Printf("Using first host %s\n", hosts[0])
	}
	return hosts[0]
}
