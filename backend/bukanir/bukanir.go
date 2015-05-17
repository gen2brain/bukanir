package main

import (
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	tmdb "github.com/amahi/go-themoviedb"
	humanize "github.com/dustin/go-humanize"
	"github.com/sqp/opensubs"
	"github.com/xrash/smetrics"
)

var (
	appName    = "bukanir-http"
	appVersion = "1.5"
)

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

type movie struct {
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
	Category     int    `json:"category"`
	Season       int    `json:"season"`
	Episode      int    `json:"episode"`
}

type summary struct {
	Id       int     `json:"id"`
	Cast     string  `json:"cast"`
	Rating   float64 `json:"rating"`
	TagLine  string  `json:"tagline"`
	Overview string  `json:"overview"`
	Runtime  int     `json:"runtime"`
	Imdb_id  string  `json:"imdbId"`
}

type subtitle struct {
	Id           string  `json:"id"`
	Title        string  `json:"title"`
	Year         string  `json:"year"`
	Release      string  `json:"release"`
	DownloadLink string  `json:"downloadLink"`
	Score        float64 `json:"score"`
}

type autocomplete struct {
	Title string `json:"title"`
	Year  string `json:"year"`
}

type language struct {
	Name      string
	ISO_639_2 string
	ID        string
}

type bySeeders []movie

func (a bySeeders) Len() int           { return len(a) }
func (a bySeeders) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a bySeeders) Less(i, j int) bool { return a[i].Seeders > a[j].Seeders }

type byScore []subtitle

func (a byScore) Len() int           { return len(a) }
func (a byScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byScore) Less(i, j int) bool { return a[i].Score > a[j].Score }

const tmdbApiKey = "YOUR_API_KEY"

var (
	reHTML   = regexp.MustCompile(`<[^>]*>`)
	reYear   = regexp.MustCompile(`(.*)(19\d{2}|20\d{2})(.*)`)
	reTitle1 = regexp.MustCompile(`(.*?)(dvdrip|xvid|dvdscr|brrip|bdrip|divx|klaxxon|hc|webrip|hdrip|camrip|hdtv|eztv|proper|x264|480p|720p|1080p|[\*\{\(\[]?[0-9]{4}).*`)
	reTitle2 = regexp.MustCompile(`(.*?)\(.*\)(.*)`)
	reSeason = regexp.MustCompile(`(?i:s|season)(\d{2})(?i:e|x|episode)(\d{2})`)
)

var categories = []string{
	"201",
	"207",
	"205",
}

var hosts = []string{
	"thepiratebay.se",
	"thepiratebay.mk",
	"thepiratebay.cd",
	"thepiratebay.lv",
}

var trackers = []string{
	"udp://tracker.publicbt.com:80/announce",
	"udp://tracker.openbittorrent.com:80/announce",
	"udp://open.demonii.com:1337/announce",
	"udp://tracker.istole.it:6969",
	"udp://tracker.coppersurfer.tk:80",
}

var languages = []language{
	{"Albanian", "alb", "29"},
	{"Arabic", "ara", "12"},
	{"Belarus", "bel", "50"},
	{"Bengali", "ben", "59"},
	{"Bosnian", "bos", "10"},
	{"Bulgarian", "bul", "33"},
	{"Catalan", "cat", "53"},
	{"Chinese", "zho", "17"},
	{"Croatian", "hrv", "38"},
	{"Czech", "ces", "7"},
	{"Danish", "dan", "24"},
	{"Dutch", "dut", "23"},
	{"English", "eng", "2"},
	{"Estonian", "est", "20"},
	{"Finnish", "fin", "31"},
	{"French", "fra", "8"},
	{"German", "ger", "5"},
	{"Greek", "gre", "16"},
	{"Hebrew", "heb", "22"},
	{"Hindi", "hin", "42"},
	{"Hungarian", "hun", "15"},
	{"Icelandic", "isl", "6"},
	{"Indonesian", "ind", "54"},
	{"Irish", "gle", "49"},
	{"Italian", "ita", "9"},
	{"Japanese", "jpn", "11"},
	{"Kazakh", "kaz", "58"},
	{"Korean", "kor", "4"},
	{"Latvian", "lav", "21"},
	{"Lithuanian", "lit", "19"},
	{"Macedonian", "mkd", "35"},
	{"Malay", "msa", "55"},
	{"Norwegian", "nor", "3"},
	{"Polish", "pol", "26"},
	{"Portuguese", "por", "32"},
	{"Romanian", "ron", "13"},
	{"Russian", "rus", "27"},
	{"Serbian", "srp", "36"},
	{"Sinhala", "sin", "56"},
	{"Slovak", "slk", "37"},
	{"Slovenian", "slv", "1"},
	{"Spanish", "spa", "28"},
	{"Swedish", "swe", "25"},
	{"Thai", "tha", "44"},
	{"Turkish", "tur", "30"},
	{"Ukrainian", "ukr", "46"},
	{"Vietnamese", "vie", "51"},
}

var movies []movie
var torrents []torrent
var subtitles []subtitle
var autocompletes []autocomplete
var movieSummary summary
var wg sync.WaitGroup

func tpbTop(category string) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tpbTop: ", r)
		}
	}()

	host := getHost()
	uri := "https://%s/top/%s"

	doc, err := getDocument(fmt.Sprintf(uri, host, category))
	if err != nil {
		log.Print("Error making TPB call: %v", err.Error())
		return
	}

	getTorrents(doc)
}

func tpbSearch(query string, page int) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tpbSearch: ", r)
		}
	}()

	host := getHost()
	uri := "https://%s/search/%s/%d/7/201,207,205"

	doc, err := getDocument(fmt.Sprintf(uri, host, url.QueryEscape(query), page))
	if err != nil {
		log.Print("Error making TPB call: %v", err.Error())
		return
	}

	getTorrents(doc)
}

func tmdbSearch(t torrent) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tmdbSearch: ", r)
		}
	}()

	md := tmdb.Init(tmdbApiKey)
	config, _ := md.GetConfig()

	var results tmdb.TmdbResponse
	if t.Category == 205 {
		results, _ = md.SearchTmdbtv(t.FormattedTitle)
	} else {
		results, _ = md.SearchMovie(t.FormattedTitle)
	}

	if results.Total_results == 0 {
		return
	}

	var res *tmdb.TmdbResult
	if t.Category == 205 {
		res = &results.Results[0]
	} else {
		res = new(tmdb.TmdbResult)
		for _, result := range results.Results {
			if result.Release_date != "" && t.Year != "" {
				tmdbYear, _ := strconv.Atoi(getYear(result.Release_date))
				torrentYear, _ := strconv.Atoi(t.Year)
				if tmdbYear == torrentYear || tmdbYear == torrentYear-1 || tmdbYear == torrentYear+1 {
					res = &result
					break
				}
			}
		}
	}

	if res.Id == 0 {
		return
	}

	var p tmdb.TmdbPoster
	if t.Category == 205 {
		p, _ = md.GetTmdbTvImages(strconv.Itoa(res.Id), t.Season)
	}

	var posterSmall, posterMedium, posterLarge, posterXLarge string
	if t.Category == 205 && len(p.Posters) > 0 {
		posterSmall = config.Images.Base_url + config.Images.Poster_sizes[0] + p.Posters[0].File_path
		posterMedium = config.Images.Base_url + config.Images.Poster_sizes[3] + p.Posters[0].File_path
		posterLarge = config.Images.Base_url + config.Images.Poster_sizes[3] + p.Posters[0].File_path
		posterXLarge = config.Images.Base_url + config.Images.Poster_sizes[4] + p.Posters[0].File_path
	} else {
		posterSmall = config.Images.Base_url + config.Images.Poster_sizes[0] + res.Poster_path
		posterMedium = config.Images.Base_url + config.Images.Poster_sizes[3] + res.Poster_path
		posterLarge = config.Images.Base_url + config.Images.Poster_sizes[3] + res.Poster_path
		posterXLarge = config.Images.Base_url + config.Images.Poster_sizes[4] + res.Poster_path
	}

	size := len(config.Images.Poster_sizes)
	if size < 5 {
		return
	}

	var title string
	var year string
	if t.Category == 205 {
		title = res.Original_name
		year = getYear(res.First_air_date)
	} else {
		title = res.Title
		year = getYear(res.Release_date)
	}

	m := movie{
		res.Id,
		title,
		year,
		posterSmall,
		posterMedium,
		posterLarge,
		posterXLarge,
		t.Size,
		t.SizeHuman,
		t.Seeders,
		t.MagnetLink,
		t.Title,
		t.Category,
		t.Season,
		t.Episode,
	}
	movies = append(movies, m)
}

func tmdbSummary(id int, category int, season int) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tmdbSummary: ", r)
		}
	}()

	md := tmdb.Init(tmdbApiKey)

	var err error
	var res tmdb.MovieMetadata

	if category == 205 {
		res, err = md.GetTmdbTvDetails(strconv.Itoa(id))
	} else {
		res, err = md.GetMovieDetails(strconv.Itoa(id))
	}
	if err != nil {
		return
	}

	if category == 205 {
		res.Credits, err = md.GetTmdbTvCredits(strconv.Itoa(id), season)
	} else {
		res.Credits, err = md.GetMovieCredits(strconv.Itoa(id))
	}
	if err != nil {
		return
	}

	res.Config, err = md.GetConfig()
	if err != nil {
		return
	}

	var imdbId string
	if category == 205 {
		ext, _ := md.GetTmdbTvExternals(strconv.Itoa(id))
		imdbId = strings.Replace(ext.Imdb_id, "tt", "", -1)
	} else {
		imdbId = strings.Replace(res.Imdb_id, "tt", "", -1)
	}

	movieSummary = summary{
		id,
		getCast(res.Credits.Cast),
		res.Vote_average,
		res.Tagline,
		res.Overview,
		res.Runtime,
		imdbId,
	}
}

func tmdbAutoComplete(query string) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tmdbAutocomplete: ", r)
		}
	}()

	md := tmdb.Init(tmdbApiKey)
	tvs, _ := md.AutoCompleteTv(query)
	movies, _ := md.AutoCompleteMovie(query)

	if tvs.Total_results+movies.Total_results == 0 {
		return
	}

	for _, movie := range movies.Results[:2] {
		year := getYear(movie.Release_date)
		if year == "" {
			continue
		}
		a := autocomplete{
			movie.Title,
			year,
		}
		autocompletes = append(autocompletes, a)
	}

	for _, tv := range tvs.Results[:2] {
		year := getYear(tv.First_air_date)
		if year == "" {
			continue
		}
		a := autocomplete{
			tv.Original_name,
			year,
		}
		autocompletes = append(autocompletes, a)
	}

	for _, movie := range movies.Results[2:7] {
		year := getYear(movie.Release_date)
		if year == "" {
			continue
		}
		a := autocomplete{
			movie.Title,
			year,
		}
		autocompletes = append(autocompletes, a)
	}

	for _, tv := range tvs.Results[2:7] {
		year := getYear(tv.First_air_date)
		if year == "" {
			continue
		}
		a := autocomplete{
			tv.Original_name,
			year,
		}
		autocompletes = append(autocompletes, a)
	}
}

func podnapisi(movie string, year string, torrentRelease string, lang string, category int, season int, episode int) {
	type Page struct {
		Current int `xml:"current"`
		Count   int `xml:"count"`
		Results int `xml:"results"`
	}

	type Subtitle struct {
		XMLName    xml.Name `xml:"subtitle"`
		Id         int      `xml:"id"`
		Pid        string   `xml:"pid"`
		Title      string   `xml:"title"`
		Year       string   `xml:"year"`
		Url        string   `xml:"url"`
		Release    string   `xml:"release"`
		TvSeason   int      `xml:"tvSeason"`
		TvEpisode  int      `xml:"tvEpisode"`
		Language   string   `xml:"language"`
		LanguageID int      `xml:"languageId"`
	}

	type Data struct {
		XMLName      xml.Name   `xml:"results"`
		Pagination   Page       `xml:"pagination"`
		SubtitleList []Subtitle `xml:"subtitle"`
	}

	l := getLanguage(lang)

	baseUrl := "http://podnapisi.net/subtitles/"
	searchUrl := baseUrl + "search/old?sXML=1&sK=%s&sY=%s&sJ=%s"

	url := fmt.Sprintf(searchUrl, url.QueryEscape(movie), year, l.ID)

	if category == 205 {
		if season != 0 {
			url = url + fmt.Sprintf("&sTS=%d&sTE=%d", season, episode)
		}
	}

	res, err := httpGet(url)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error unmarshalling response: %v\n", err)
	}

	var r Data
	err = xml.Unmarshal(body, &r)
	if err != nil {
		log.Printf("Error unmarshalling response: %v\n", err)
	}

	for _, s := range r.SubtitleList {
		rel := strings.Fields(s.Release)
		var subtitleRelease string
		if len(rel) > 0 {
			subtitleRelease = rel[0]
		} else {
			continue
		}

		lid, _ := strconv.Atoi(l.ID)
		if s.LanguageID != lid {
			continue
		}

		score := compareRelease(torrentRelease, subtitleRelease)
		if score < 0.7 {
			continue
		}

		if category == 205 {
			if season != s.TvSeason || episode != s.TvEpisode {
				continue
			}
		}

		downloadLink := fmt.Sprintf(baseUrl+"%s/download", s.Pid)

		s := subtitle{strconv.Itoa(s.Id), s.Title, s.Year, subtitleRelease, downloadLink, score}
		subtitles = append(subtitles, s)
	}
}

func titlovi(movie string, torrentRelease string, category int, season int, episode int) {
	searchUrl := "http://en.titlovi.com/subtitles/subtitles.aspx?subtitle=%s"
	downloadUrl := "http://titlovi.com/downloads/default.ashx?type=1&mediaid=%s"

	url := fmt.Sprintf(searchUrl, url.QueryEscape(movie))

	doc, err := getDocument(url)
	if err != nil {
		log.Fatal(err)
	}

	nodes := doc.Find(`li.listing`)
	divs := nodes.Find(`div.title.c1`)

	if divs.Length() == 0 {
		return
	}

	reNum := regexp.MustCompile(`[^0-9]`)

	parseHTML := func(i int, div *goquery.Selection) {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Print("Recovered in parseHTML: ", r)
			}
		}()

		link := div.Find(`a`).First()
		href, _ := link.Attr("href")

		title := getTitle(strings.TrimSpace(link.Text()))

		subtitleYear := div.Find(`span.year`).First().Text()
		year := getYear(subtitleYear)

		subtitleRelease := div.Find(`span.release`).First().Text()

		split := strings.Split(href, "-")
		id := split[len(split)-1]
		id = reNum.ReplaceAllString(id, "")

		downloadLink := fmt.Sprintf(downloadUrl, id)

		score := compareRelease(torrentRelease, subtitleRelease)
		if score < 0.7 {
			return
		}

		if category == 205 {
			rs, _ := strconv.Atoi(getSeason(subtitleRelease))
			re, _ := strconv.Atoi(getEpisode(subtitleRelease))
			if season != rs || episode != re {
				return
			}
		}

		s := subtitle{id, title, year, subtitleRelease, downloadLink, score}
		subtitles = append(subtitles, s)
	}

	wg.Add(divs.Length())
	divs.Each(func(i int, s *goquery.Selection) {
		go parseHTML(i, s)
	})
	wg.Wait()
}

func opensubtitles(movie string, imdbId string, year string, torrentRelease string, lang string, category int, season int, episode int) {
	const OPENSUBTITLE_USER_AGENT = "OSTestUserAgent"

	l := getLanguage(lang)

	query := opensubs.NewQuery(OPENSUBTITLE_USER_AGENT)
	query.AddImdb(imdbId, l.ISO_639_2)

	if category == 205 {
		if season != 0 {
			query.AddSeason(strconv.Itoa(season))
			query.AddEpisode(strconv.Itoa(episode))
		}
	}

	query.Search()
	defer query.Logout()

	_, byimdb := query.Get(-1)
	for _, bylang := range byimdb {
		for _, list := range bylang {
			for _, sub := range list {

				if sub.SubLanguageID != l.ISO_639_2 {
					continue
				}

				score := compareRelease(torrentRelease, sub.MovieReleaseName)
				if score < 0.7 {
					continue
				}

				if category == 205 {
					subSeason, _ := strconv.Atoi(sub.SeriesSeason)
					subEpisode, _ := strconv.Atoi(sub.SeriesEpisode)
					if season != subSeason || episode != subEpisode {
						continue
					}
				}

				s := subtitle{sub.IDSubtitleFile, sub.MovieName, sub.MovieYear, sub.MovieReleaseName, sub.ZipDownloadLink, score}
				subtitles = append(subtitles, s)
			}
		}
	}
}

func compareRelease(torrentRelease string, subtitleRelease string) float64 {
	torrentRelease = strings.Replace(torrentRelease, ".", " ", -1)
	torrentRelease = strings.Replace(torrentRelease, "-", " ", -1)
	subtitleRelease = strings.Replace(subtitleRelease, ".", " ", -1)
	subtitleRelease = strings.Replace(subtitleRelease, "-", " ", -1)
	return smetrics.Jaro(torrentRelease, subtitleRelease)
}

func httpGet(uri string) (*http.Response, error) {
	jar, _ := cookiejar.New(nil)
	timeout := time.Duration(30 * time.Second)

	dialTimeout := func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, timeout)
	}

	transport := http.Transport{
		Dial:            dialTimeout,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	httpClient := http.Client{
		Jar:       jar,
		Transport: &transport,
	}

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Close = true
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:36.0) Gecko/20100101 Firefox/36.0")

	res, err := httpClient.Do(req)
	if err != nil || res == nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, nil
	}

	return res, nil
}

func getDocument(uri string) (*goquery.Document, error) {
	res, err := httpGet(uri)
	if err != nil {
		log.Printf("Error httpGet %s: %v", uri, err.Error())
		return nil, err
	}

	if res == nil {
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Printf("Error NewDocumentFromResponse %s: %v", uri, err.Error())
		return nil, err
	}

	if doc == nil {
		return nil, nil
	}

	return doc, nil
}

func getTorrents(doc *goquery.Document) {
	divs := doc.Find(`div.detName`)

	if divs.Length() == 0 {
		return
	}

	var wgp sync.WaitGroup

	parseHTML := func(i int, s *goquery.Selection) {
		defer wgp.Done()

		parent := s.Parent()
		prev := parent.Prev().First()

		title := s.Find(`a.detLink`).Text()
		magnet, _ := parent.Find(`a[title="Download this torrent using magnet"]`).Attr(`href`)
		desc := parent.Find(`font.detDesc`).Text()
		seeders, _ := strconv.Atoi(parent.Next().Text())

		c, _ := prev.Find(`a[title="More from this category"]`).Last().Attr(`href`)
		category, _ := strconv.Atoi(strings.Replace(c, "/browse/", "", -1))

		if seeders == 0 {
			return
		}

		var size uint64
		var sizeHuman string
		parts := strings.Split(desc, ", ")
		if len(parts) > 1 {
			size, _ = humanize.ParseBytes(strings.Split(parts[1], " ")[1])
			sizeHuman = humanize.IBytes(size)
		}

		season, _ := strconv.Atoi(getSeason(title))
		episode, _ := strconv.Atoi(getEpisode(title))

		t := torrent{
			title,
			getTitle(title),
			boostMagnet(magnet),
			getYear(title),
			size,
			sizeHuman,
			seeders,
			category,
			season,
			episode,
		}

		torrents = append(torrents, t)
	}

	wgp.Add(divs.Length())
	divs.Each(func(i int, s *goquery.Selection) {
		go parseHTML(i, s)
	})
	wgp.Wait()
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

	title = reSeason.ReplaceAllString(title, "")

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

func getSeason(torrentTitle string) string {
	season := ""
	re := reSeason.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		season = re[0][1]
	}
	return season
}

func getEpisode(torrentTitle string) string {
	episode := ""
	re := reSeason.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		episode = re[0][2]
	}
	return episode
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

func getLanguage(name string) *language {
	lang := new(language)
	for _, l := range languages {
		if strings.ToLower(name) == strings.ToLower(l.Name) {
			lang = &l
			break
		}
	}
	return lang
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

func saveCache(key string, data []byte, tmpDir string) {
	file := filepath.Join(tmpDir, key+".json")
	err := ioutil.WriteFile(file, data, 0644)
	if err != nil {
		log.Print("Error writing cache file: %v", err.Error())
	}
}

func getCache(key string, tmpDir string) []byte {
	file := filepath.Join(tmpDir, key+".json")
	info, err := os.Stat(file)
	if err != nil {
		return nil
	}
	mtime := info.ModTime().Unix()
	if time.Now().Unix()-mtime > 43200 {
		return nil
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Print("Error reading cache file: %v", err.Error())
		return nil
	}
	return data
}

func Category(category string, limit int, force int, tmpDir string) (string, error) {
	if force != 1 {
		cache := getCache(category, tmpDir)
		if cache != nil {
			return string(cache[:]), nil
		}
	}

	torrents = make([]torrent, 0)

	wg.Add(1)
	go tpbTop(category)
	wg.Wait()

	if limit > 0 {
		if limit > len(torrents) {
			limit = len(torrents)
		}
		torrents = torrents[0:limit]
	}

	movies = make([]movie, 0)
	wg.Add(len(torrents))
	for _, torrent := range torrents {
		go tmdbSearch(torrent)
	}
	wg.Wait()

	sort.Sort(bySeeders(movies))
	js, err := json.MarshalIndent(movies, "", "    ")
	if err != nil {
		return "empty", err
	}

	if len(movies) > 0 {
		saveCache(category, js, tmpDir)
	}

	return string(js[:]), nil
}

func Search(query string, limit int) (string, error) {
	torrents = make([]torrent, 0)

	wg.Add(3)
	go tpbSearch(query, 0)
	go tpbSearch(query, 1)
	go tpbSearch(query, 2)
	wg.Wait()

	if limit > 0 {
		if limit > len(torrents) {
			limit = len(torrents)
		}
		torrents = torrents[0:limit]
	}

	movies = make([]movie, 0)
	wg.Add(len(torrents))
	for _, torrent := range torrents {
		go tmdbSearch(torrent)
	}
	wg.Wait()

	sort.Sort(bySeeders(movies))
	js, err := json.MarshalIndent(movies, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func Summary(id int, category int, season int) (string, error) {
	movieSummary = summary{}
	wg.Add(1)
	go tmdbSummary(id, category, season)
	wg.Wait()

	js, err := json.MarshalIndent(movieSummary, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func Subtitle(movie string, year string, release string, language string, category int, season int, episode int, imdbID string) (string, error) {
	subtitles = make([]subtitle, 0)
	language = strings.ToLower(language)

	podnapisi(movie, year, release, language, category, season, episode)
	opensubtitles(movie, imdbID, year, release, language, category, season, episode)

	if language == "serbian" || language == "croation" || language == "bosnian" {
		titlovi(movie, release, category, season, episode)
	}

	if len(subtitles) == 0 && language != "english" {
		podnapisi(movie, year, release, "english", category, season, episode)
		opensubtitles(movie, imdbID, year, release, "english", category, season, episode)
	}

	sort.Sort(byScore(subtitles))

	js, err := json.MarshalIndent(subtitles, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func AutoComplete(query string, limit int) (string, error) {
	autocompletes = make([]autocomplete, 0)
	wg.Add(1)
	go tmdbAutoComplete(query)
	wg.Wait()

	if limit > 0 {
		if limit > len(autocompletes) {
			limit = len(autocompletes)
		}
		autocompletes = autocompletes[0:limit]
	}

	js, err := json.MarshalIndent(autocompletes, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func setServer(w http.ResponseWriter) {
	w.Header().Set("Server", fmt.Sprintf("%s/%s", appName, appVersion))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	setServer(w)
	http.Error(w, "403 Forbidden", http.StatusForbidden)
	return
}

func handleCategory(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	category := r.FormValue("c")
	limit, _ := strconv.Atoi(r.FormValue("l"))
	force, _ := strconv.Atoi(r.FormValue("f"))
	tmpdir := r.FormValue("t")

	if isValidCategory(category) {
		js, err := Category(category, limit, force, tmpdir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(js))
	} else {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	query := r.FormValue("q")
	limit, _ := strconv.Atoi(r.FormValue("l"))

	js, err := Search(query, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(js))
}

func handleSummary(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	id, _ := strconv.Atoi(r.FormValue("i"))
	category, _ := strconv.Atoi(r.FormValue("c"))
	season, _ := strconv.Atoi(r.FormValue("s"))

	js, err := Summary(id, category, season)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(js))
}

func handleSubtitle(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	movie := r.FormValue("m")
	year := r.FormValue("y")
	release := r.FormValue("r")
	language := r.FormValue("l")
	category, _ := strconv.Atoi(r.FormValue("c"))
	season, _ := strconv.Atoi(r.FormValue("s"))
	episode, _ := strconv.Atoi(r.FormValue("e"))
	imdbId := r.FormValue("i")

	js, err := Subtitle(movie, year, release, language, category, season, episode, imdbId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(js))
}

func handleAutoComplete(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	query := r.FormValue("q")
	limit, _ := strconv.Atoi(r.FormValue("l"))

	js, err := AutoComplete(query, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(js))
}

func main() {
	bind := flag.String("bind", ":7314", "Bind address")
	flag.Parse()

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/search", handleSearch)
	http.HandleFunc("/category", handleCategory)
	http.HandleFunc("/summary", handleSummary)
	http.HandleFunc("/subtitle", handleSubtitle)
	http.HandleFunc("/autocomplete", handleAutoComplete)

	l, err := net.Listen("tcp4", *bind)
	if err != nil {
		log.Fatal(err)
	}
	http.Serve(l, nil)
	defer l.Close()
}
