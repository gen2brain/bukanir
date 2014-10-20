package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.google.com/p/go.net/html"
	"github.com/PuerkitoBio/goquery"
	tmdb "github.com/amahi/go-themoviedb"
	humanize "github.com/dustin/go-humanize"
	"github.com/xrash/smetrics"
)

var (
	appName    = "bukanir-http"
	appVersion = "1.1"
)

type Torrent struct {
	Title          string
	FormattedTitle string
	MagnetLink     string
	Year           string
	Size           uint64
	SizeHuman      string
	Seeders        int
}

type Movie struct {
	Id           int    `json:"id"`
	Title        string `json:"title"`
	Year         string `json:"year"`
	PosterSmall  string `json:"posterSmall"`
	PosterMedium string `json:"posterMedium"`
	PosterLarge  string `json:"posterLarge"`
	PosterXLarge string `json:"posterXLarge"`
	Size         uint64 `json:"size"`
	SizeHuman    string `json:"sizeHuman"`
	Seeders      int    `json:"seeders"`
	MagnetLink   string `json:"magnetLink"`
	Release      string `json:"release"`
}

type Summary struct {
	Id       int     `json:"id"`
	Cast     string  `json:"cast"`
	Rating   float64 `json:"rating"`
	TagLine  string  `json:"tagline"`
	Overview string  `json:"overview"`
	Runtime  int     `json:"runtime"`
}

type Subtitle struct {
	Id	         string  `json:"id"`
	Title        string  `json:"title"`
	Year         string  `json:"year"`
	Release      string  `json:"release"`
	DownloadLink string  `json:"downloadLink"`
	Score        float64 `json:"score"`
}

type Release struct {
	Name  string
	Score float64
}

type BySeeders []Movie

func (a BySeeders) Len() int           { return len(a) }
func (a BySeeders) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySeeders) Less(i, j int) bool { return a[i].Seeders > a[j].Seeders }

type ReleaseByScore []Release

func (a ReleaseByScore) Len() int           { return len(a) }
func (a ReleaseByScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ReleaseByScore) Less(i, j int) bool { return a[i].Score > a[j].Score }

type ByScore []Subtitle

func (a ByScore) Len() int           { return len(a) }
func (a ByScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByScore) Less(i, j int) bool { return a[i].Score > a[j].Score }

var tpbTopUrl string = "http://%s/top/%s"
var tpbSearchUrl string = "http://%s/search/%s/0/7/201,202"
const tmdbApiKey = "YOUR_API_KEY"

var (
	reYear   = regexp.MustCompile(`(.*)(19\d{2}|20\d{2})(.*)`)
	reTitle1 = regexp.MustCompile(`(.*?)(dvdrip|xvid|dvdscr|brrip|bdrip|divx|klaxxon|hc|webrip|hdrip|camrip|hdtv|eztv|proper|720p|1080p|[\*\{\(\[]?[0-9]{4}).*`)
	reTitle2 = regexp.MustCompile(`(.*?)\(.*\)(.*)`)
)

var categories = []string{
	"201",
	"202",
	"207",
}

var hosts = []string{
	"thepiratebay.se",
	"thepiratebay.mg",
	"thepiratebay.si",
	"thepiratebay.je",
	"pirateproxy.net",
}

var trackers = []string{
	"udp://tracker.publicbt.com:80/announce",
	"udp://tracker.openbittorrent.com:80/announce",
	"udp://open.demonii.com:1337/announce",
	"udp://tracker.istole.it:6969",
	"udp://tracker.coppersurfer.tk:80",
}

var movies []Movie
var summary Summary
var torrents []Torrent
var releases []Release
var subtitles []Subtitle
var wg sync.WaitGroup

var cacheDir = flag.String("cachedir", "/tmp", "Cache directory")

func TPBTop(category string) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in TPBTop: ", r)
		}
	}()

	host := getHost()
	res, err := httpGet(fmt.Sprintf(tpbTopUrl, host, category))
	if err != nil {
		log.Print("Error making TPB call: %v", err.Error())
		return
	}

	doc, err := html.Parse(res.Body)
	if err != nil {
		log.Print("Error parsing TPB html: %v", err.Error())
		return
	}

	loopDOM(doc)
}

func TPBSearch(query string) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in TPBSearch: ", r)
		}
	}()

	host := getHost()
	res, err := httpGet(fmt.Sprintf(tpbSearchUrl, host, url.QueryEscape(query)))
	if err != nil {
		log.Print("Error making TPB call: %v", err.Error())
		return
	}

	doc, err := html.Parse(res.Body)
	if err != nil {
		log.Print("Error parsing TPB html: %v", err.Error())
		return
	}

	loopDOM(doc)
}

func TMDBMovie(torrent Torrent) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in TMDBMovie: ", r)
		}
	}()

	md := tmdb.Init(tmdbApiKey)
	config, _ := md.GetConfig()

	results, _ := md.SearchMovie(torrent.FormattedTitle)
	if results.Total_results == 0 {
		return
	}

	res := new(tmdb.TmdbResult)
	for _, result := range results.Results {
		if result.Release_date != "" && torrent.Year != "" {
			tmdbYear, _ := strconv.Atoi(getYear(result.Release_date))
			torrentYear, _ := strconv.Atoi(torrent.Year)
			if tmdbYear == torrentYear || tmdbYear == torrentYear-1 || tmdbYear == torrentYear+1 {
				res = &result
				break
			}
		}
	}

	if res.Id == 0 {
		return
	}

	movie := Movie{
		res.Id,
		res.Title,
		getYear(res.Release_date),
		config.Images.Base_url + config.Images.Poster_sizes[0] + res.Poster_path,
		config.Images.Base_url + config.Images.Poster_sizes[3] + res.Poster_path,
		config.Images.Base_url + config.Images.Poster_sizes[3] + res.Poster_path,
		config.Images.Base_url + config.Images.Poster_sizes[4] + res.Poster_path,
		torrent.Size,
		torrent.SizeHuman,
		torrent.Seeders,
		torrent.MagnetLink,
		torrent.Title,
	}
	movies = append(movies, movie)
}

func TMDBSummary(id int) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in TMDBSummary: ", r)
		}
	}()

	md := tmdb.Init(tmdbApiKey)

	res, err := md.GetMovieDetails(strconv.Itoa(id))
	if err != nil {
		return
	}
	res.Credits, err = md.GetMovieCredits(strconv.Itoa(id))
	if err != nil {
		return
	}
	res.Config, err = md.GetConfig()
	if err != nil {
		return
	}

	summary = Summary{
		id,
		getCast(res.Credits.Cast),
		res.Vote_average,
		res.Tagline,
		res.Overview,
		res.Runtime,
	}
}

func Podnapisi(movie string, year string, torrentRelease string, lang string) {
	baseUrl := "http://www.podnapisi.net"
	searchUrl := baseUrl + "/en/ppodnapisi/search?sT=-1&sK=%s&sJ=%s&sY=%s&sAKA=0&sS=downloads&sO=desc"

	url := fmt.Sprintf(searchUrl, url.QueryEscape(movie), lang, year)

	doc, err := goquery.NewDocument(url)
	if err != nil {
		log.Fatal(err)
	}

	nodes := doc.Find(`table.list.first_column_title`)
	trs := nodes.Find(`tr[class=" "]`)

	length := trs.Length()
	if length == 0 {

	}

	getSubtitle := func(tr *goquery.Selection) {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Print("Recovered in getSubtitle: ", r)
			}
		}()

		subtitlePageLink := tr.Find(`a.subtitle_page_link`).First()
		subtitleYear := subtitlePageLink.Find(`b`).First().Text()

		rl, _ := tr.Find(`span.release`).Attr(`html_title`)
		subtitleReleases := strings.Split(rl, "<br/>")

		releases = make([]Release, 0)
		for _, rel := range subtitleReleases {
			if rel == "" {
				continue
			}
			score := compareRelease(torrentRelease, rel)
			r := Release{rel, score}
			releases = append(releases, r)
		}
		sort.Sort(ReleaseByScore(releases))

		var release Release
		if len(releases) > 0 {
			release = releases[0]
		} else {
			release = Release{"", 0}
		}

		subtitleHref, _ := subtitlePageLink.Attr(`href`)
		downloadUrl := baseUrl + subtitleHref
		downloadDoc, err := goquery.NewDocument(downloadUrl)
		if err != nil {
			log.Fatal(err)
		}

		downloadHref := downloadDoc.Find(`a.button.big.download`).First()
		href, _ := downloadHref.Attr(`href`)
		downloadLink := baseUrl + strings.Replace(href, "predownload", "download", -1)

		split := strings.Split(href, "/")
		id := split[len(split)-1]
		title := strings.Replace(subtitlePageLink.Text(), subtitleYear, "", -1)
		year := getYear(subtitleYear)

		subtitle := Subtitle{id, title, year, release.Name, downloadLink, release.Score}
		subtitles = append(subtitles, subtitle)
	}

	wg.Add(len(trs.Nodes))
	for i := range trs.Nodes {
		tr := trs.Eq(i)
		go getSubtitle(tr)
	}
	wg.Wait()
}

func Titlovi(movie string, torrentRelease string) {
	searchUrl := "http://en.titlovi.com/subtitles/subtitles.aspx?subtitle=%s"
	downloadUrl := "http://titlovi.com/downloads/default.ashx?type=1&mediaid=%s"

	url := fmt.Sprintf(searchUrl, url.QueryEscape(movie))

	doc, err := goquery.NewDocument(url)
	if err != nil {
		log.Fatal(err)
	}

	nodes := doc.Find(`li.listing`)
	divs := nodes.Find(`div.title.c1`)

	length := divs.Length()
	if length == 0 {

	}

	reNum := regexp.MustCompile(`[^0-9]`)

	for i := range divs.Nodes {
		div := divs.Eq(i)

		link := div.Find(`a`).First()
		href, _ := link.Attr("href")

		title := strings.TrimSpace(link.Text())

		subtitleYear := div.Find(`span.year`).First().Text()
		year := getYear(subtitleYear)

		subtitleRelease := div.Find(`span.release`).First().Text()
		release := strings.TrimSpace(subtitleRelease)

		score := compareRelease(torrentRelease, release)

		split := strings.Split(href, "-")
		id := split[len(split) - 1]
		id = reNum.ReplaceAllString(id, "")

		downloadLink := fmt.Sprintf(downloadUrl, id)

		subtitle := Subtitle{id, title, year, release, downloadLink, score}
		subtitles = append(subtitles, subtitle)
	}
}

func compareRelease(torrentRelease string, subtitleRelease string) float64 {
	torrentRelease = strings.Replace(torrentRelease, ".", " ", -1)
	torrentRelease = strings.Replace(torrentRelease, "-", " ", -1)
	subtitleRelease = strings.Replace(subtitleRelease, ".", " ", -1)
	subtitleRelease = strings.Replace(subtitleRelease, "-", " ", -1)
	return smetrics.Jaro(torrentRelease, subtitleRelease)
}

func httpGet(url string) (*http.Response, error) {
	timeout := time.Duration(10 * time.Second)

	dialTimeout := func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, timeout)
	}

	transport := http.Transport{
		Dial: dialTimeout,
	}

	httpClient := http.Client{
		Transport: &transport,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:25.0) Gecko/20100101 Firefox/25.0")
	return httpClient.Do(req)
}

func loopDOM(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "tbody" {
		extractTorrents(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		loopDOM(c)
	}
}

func extractTorrents(n *html.Node) {
	torrents = make([]Torrent, 0)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tr" {
			var torrent Torrent
			err := getTorrent(c, &torrent)
			if err == nil {
				torrent.Year = getYear(torrent.Title)
				torrent.FormattedTitle = getTitle(torrent.Title)
				torrent.MagnetLink = boostMagnet(torrent.MagnetLink)
				torrents = append(torrents, torrent)
			}
		}
	}
}

func getTorrent(n *html.Node, t *Torrent) error {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if n.Data == "a" && a.Key == "href" && a.Val[:6] == "magnet" {
				t.MagnetLink = a.Val
			} else if n.Data == "a" && a.Key == "href" && a.Val[:9] == "/torrent/" {
				if t.Title == "" {
					t.Title = getNodeText(n)
				}
			} else if n.Data == "font" && a.Key == "class" && a.Val == "detDesc" {
				parts := strings.Split(getNodeText(n), ", ")
				if len(parts) > 1 {
					s, _ := humanize.ParseBytes(strings.Split(parts[1], " ")[1])
					t.Size = s
					t.SizeHuman = humanize.IBytes(s)
				}
			} else if n.Data == "td" && a.Key == "align" && a.Val == "right" {
				if t.Seeders == 0 {
					t.Seeders, _ = strconv.Atoi(getNodeText(n))
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		getTorrent(c, t)
	}
	return nil
}

func getNodeText(n *html.Node) string {
	for a := n.FirstChild; a != nil; a = a.NextSibling {
		if a.Type == html.TextNode {
			return strings.TrimSpace(a.Data)
		}
	}
	return ""
}

func getTitle(torrentTitle string) string {
	title := strings.ToLower(torrentTitle)
	title = strings.Replace(title, ".", " ", -1)
	title = strings.Replace(title, "-", "", -1)

	re1 := reTitle1.FindAllStringSubmatch(title, -1)
	if len(re1) > 0 {
		title = re1[0][1]
	}

	re2 := reTitle2.FindAllStringSubmatch(title, -1)
	if len(re2) > 0 {
		title = re2[0][1]
	}

	title = strings.Replace(title, "(", "", -1)
	title = strings.Replace(title, ")", "", -1)

	return strings.Trim(title, " ")
}

func getYear(torrentTitle string) string {
	title := ""
	re := reYear.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		title = re[0][2]
	}
	return title
}

func getHost() string {
	for _, host := range hosts {
		_, err := net.Dial("tcp", host+":80")
		if err == nil {
			return host
		}
	}
	return hosts[0]
}

func getCast(res []tmdb.TmdbCast) string {
	cast := ""
	castLen := len(res)
	if castLen >= 4 {
		for n, c := range res[0:3] {
			cast += c.Name
			if n != 2 {
				cast += ", "
			} else {
				cast += "..."
			}
		}
	} else if castLen == 3 {
		for n, c := range res[0:2] {
			cast += c.Name
			if n != 2 {
				cast += ", "
			}
		}
	} else if castLen == 2 {
		cast += res[0].Name + ", "
		cast += res[1].Name
	} else {
		cast += res[0].Name
	}
	return cast
}

func isValidCategory(category string) bool {
	for _, cat := range categories {
		if cat == category {
			return true
		}
	}
	return false
}

func boostMagnet(magnet string) string {
	for _, tracker := range trackers {
		magnet += "&tr=" + url.QueryEscape(tracker)
	}
	return magnet
}

func saveCache(key string, data []byte) {
	file := filepath.Join(*cacheDir, key+".json")
	err := ioutil.WriteFile(file, data, 0644)
	if err != nil {
		log.Print("Error writing cache file: %v", err.Error())
	}
}

func getCache(key string) []byte {
	file := filepath.Join(*cacheDir, key+".json")
	info, err := os.Stat(file)
	if err != nil {
		return nil
	}
	mtime := info.ModTime().Unix()
	if time.Now().Unix()-mtime > 86400 {
		return nil
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Print("Error reading cache file: %v", err.Error())
		return nil
	}
	return data
}

func setServer(w http.ResponseWriter) {
	w.Header().Set("Server", fmt.Sprintf("%s/%s", appName, appVersion))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	setServer(w)
	if r.URL.Path[1:] != "" {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}
	w.WriteHeader(200)
}

func handleCategory(w http.ResponseWriter, r *http.Request) {
	setServer(w)
	path := html.EscapeString(r.URL.Path[1:])
	path = strings.TrimRight(path, "/")
	paths := strings.Split(path, "/")

	if len(paths) < 2 {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}

	category := paths[1]
	if isValidCategory(category) {
		refresh := false
		if len(paths) == 4 {
			force, _ := strconv.Atoi(paths[3])
			if force == 1 {
				refresh = true
			}
		}

		cache := getCache(category)
		if cache != nil && !refresh {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Write(cache)
			return
		}

		wg.Add(1)
		go TPBTop(category)
		wg.Wait()

		if len(paths) >= 3 {
			limit, _ := strconv.Atoi(paths[2])
			if limit > len(torrents) {
				limit = len(torrents)
			}
			torrents = torrents[0:limit]
		}

		movies = make([]Movie, 0)
		wg.Add(len(torrents))
		for _, torrent := range torrents {
			go TMDBMovie(torrent)
		}
		wg.Wait()

		sort.Sort(BySeeders(movies))
		js, err := json.MarshalIndent(movies, "", "    ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		saveCache(category, js)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(js)
	} else {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	setServer(w)
	path := html.EscapeString(r.URL.Path[1:])
	path = strings.TrimRight(path, "/")
	paths := strings.Split(path, "/")

	if len(paths) < 2 {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}

	query := paths[1]
	wg.Add(1)
	go TPBSearch(query)
	wg.Wait()

	if len(paths) == 3 {
		limit, _ := strconv.Atoi(paths[2])
		if limit > len(torrents) {
			limit = len(torrents)
		}
		torrents = torrents[0:limit]
	}

	movies = make([]Movie, 0)
	wg.Add(len(torrents))
	for _, torrent := range torrents {
		go TMDBMovie(torrent)
	}
	wg.Wait()

	sort.Sort(BySeeders(movies))
	js, err := json.MarshalIndent(movies, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(js)
}

func handleSummary(w http.ResponseWriter, r *http.Request) {
	setServer(w)
	path := html.EscapeString(r.URL.Path[1:])
	path = strings.TrimRight(path, "/")
	paths := strings.Split(path, "/")

	if len(paths) < 2 {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}

	id, _ := strconv.Atoi(paths[1])
	wg.Add(1)
	go TMDBSummary(id)
	wg.Wait()

	js, err := json.MarshalIndent(summary, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(js)
}

func handleSubtitle(w http.ResponseWriter, r *http.Request) {
	setServer(w)
	path := html.EscapeString(r.URL.Path[1:])
	path = strings.TrimRight(path, "/")
	paths := strings.Split(path, "/")

	if len(paths) < 5 {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}

	subtitles = make([]Subtitle, 0)
	Podnapisi(paths[1], paths[2], paths[3], paths[4])
	if paths[4] == "36" {
		Titlovi(paths[1], paths[3])
	}
	sort.Sort(ByScore(subtitles))

	js, err := json.MarshalIndent(subtitles, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(js)
}

func main() {
	bind := flag.String("bind", ":7314", "Bind address")
	flag.Parse()

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/search/", handleSearch)
	http.HandleFunc("/category/", handleCategory)
	http.HandleFunc("/summary/", handleSummary)
	http.HandleFunc("/subtitle/", handleSubtitle)

    l, err := net.Listen("tcp4", *bind)
    if err != nil {
        log.Fatal(err)
    }
    http.Serve(l, nil)
	defer l.Close()
}
