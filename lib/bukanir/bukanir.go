package bukanir

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	t2http "github.com/gen2brain/bukanir/lib/torrent2http"
	"github.com/gen2brain/vidextr"
)

type TMovie struct {
	Id           int    `json:"id"`
	Title        string `json:"title"`
	Year         string `json:"year"`
	PosterSmall  string `json:"posterSmall"`
	PosterMedium string `json:"posterMedium"`
	PosterLarge  string `json:"posterLarge"`
	PosterXLarge string `json:"posterXLarge"`
	Size         int64  `json:"size"`
	SizeHuman    string `json:"sizeHuman"`
	Seeders      int    `json:"seeders"`
	MagnetLink   string `json:"magnetLink"`
	Release      string `json:"release"`
	Category     int    `json:"category"`
	Season       int    `json:"season"`
	Episode      int    `json:"episode"`
	Quality      string `json:"quality"`
}

type TSummary struct {
	Id       int      `json:"id"`
	Cast     []string `json:"cast"`
	Genre    []string `json:"genre"`
	Video    string   `json:"video"`
	Director string   `json:"director"`
	Rating   float64  `json:"rating"`
	TagLine  string   `json:"tagline"`
	Overview string   `json:"overview"`
	Runtime  int      `json:"runtime"`
	ImdbId   string   `json:"imdbId"`
}

type TSubtitle struct {
	Id           string  `json:"id"`
	Title        string  `json:"title"`
	Year         string  `json:"year"`
	Release      string  `json:"release"`
	DownloadLink string  `json:"downloadLink"`
	Score        float64 `json:"score"`
}

type TItem struct {
	Title string `json:"title"`
	Year  string `json:"year"`
}

type TGenre struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type tTorrent struct {
	Title          string
	FormattedTitle string
	MagnetLink     string
	Year           string
	Size           int64
	SizeHuman      string
	Seeders        int
	Category       int
	Season         int
	Episode        int
}

type TConfig t2http.Config
type TStatus t2http.SessionStatus
type TLsInfo t2http.LsInfo
type TFileInfo t2http.FileStatusInfo

type bySeeders []TMovie

func (a bySeeders) Len() int           { return len(a) }
func (a bySeeders) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a bySeeders) Less(i, j int) bool { return a[i].Seeders > a[j].Seeders }

type byScore []TSubtitle

func (a byScore) Len() int           { return len(a) }
func (a byScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byScore) Less(i, j int) bool { return a[i].Score > a[j].Score }

type byTSeeders []tTorrent

func (a byTSeeders) Len() int           { return len(a) }
func (a byTSeeders) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTSeeders) Less(i, j int) bool { return a[i].Seeders > a[j].Seeders }

const (
	Version = "2.3"

	tmdbApiUrl = "http://api.themoviedb.org/3"
	tmdbApiKey = "YOUR_API_KEY"

	opensubsUser      = ""
	opensubsPassword  = ""
	opensubsUserAgent = "OSTestUserAgent"
)

const (
	CategoryMovies   = 201
	CategoryHDmovies = 207
	CategoryTV       = 205
	CategoryHDTV     = 208
)

var Categories = []int{
	CategoryMovies,
	CategoryHDmovies,
	CategoryTV,
	CategoryHDTV,
}

var TpbHosts = []string{
	"thepiratebay.org",
	"thepiratebay.mk",
	"thepbproxy.site",
	"thepiratebay.lv",
	"proxybay.site",
}

var EztvHosts = []string{
	"eztv.ag",
	"eztv.tf",
	"eztv.yt",
}

var (
	movies        []TMovie
	subtitles     []TSubtitle
	autocompletes []TItem
	popular       []TItem
	toprated      []TItem
	genres        []TGenre
	bygenre       []TMovie
	torrents      []tTorrent
	details       TSummary

	wg, wgt, wgs sync.WaitGroup
	verbose      bool
	throttle     chan int
	cancelchan   chan bool
)

var (
	ctx, cancel = context.WithCancel(context.TODO())

	clientFast *http.Client = &http.Client{
		Transport: &http.Transport{
			Dial:                func(network, addr string) (net.Conn, error) { return net.DialTimeout(network, addr, 5*time.Second) },
			TLSHandshakeTimeout: 5 * time.Second,
			MaxIdleConnsPerHost: 10,
		},
		Timeout: 5 * time.Second,
	}

	clientSlow *http.Client = &http.Client{
		Transport: &http.Transport{
			Dial:                func(network, addr string) (net.Conn, error) { return net.DialTimeout(network, addr, 5*time.Second) },
			TLSHandshakeTimeout: 10 * time.Second,
			MaxIdleConnsPerHost: 15,
		},
		Timeout: 35 * time.Second,
	}
)

func Category(category int, limit int, force int, cacheDir string, cacheDays int64, tpbHost string) (string, error) {
	if force != 1 {
		cache := getCache(strconv.Itoa(category), cacheDir, cacheDays)
		if cache != nil {
			return string(cache[:]), nil
		}
	}

	cancelchan = make(chan bool)
	ctx, cancel = context.WithCancel(context.TODO())
	torrents = make([]tTorrent, 0)

	if tpbHost == "" {
		tpbHost = getTpbHost()
	}

	wgt.Add(1)
	go tpbTop(category, tpbHost)
	wgt.Wait()

	if limit > 0 {
		if limit > len(torrents) {
			limit = len(torrents)
		}
		torrents = torrents[0:limit]
	}

	if verbose {
		log.Printf("BUK: Total torrents: %d\n", len(torrents))
	}

	movies = make([]TMovie, 0)

	md := NewTmdb(tmdbApiKey)
	config, err := md.GetConfig()
	if err != nil {
		log.Printf("ERROR: TMDB GetConfig: %s\n", err.Error())
		return "empty", err
	}

	throttle = make(chan int, 10)

	if len(torrents) > 0 {
		for _, torrent := range torrents {
			throttle <- 1
			wg.Add(1)
			if torrent.Category == CategoryTV || torrent.Category == CategoryHDTV {
				go tmdbSearchTv(torrent, config)
			} else {
				go tmdbSearchMovie(torrent, config)
			}
		}
		wg.Wait()
	}

	if verbose {
		log.Printf("BUK: Total movies: %d\n", len(movies))
	}

	sort.Sort(bySeeders(movies))
	js, err := json.MarshalIndent(movies, "", "    ")
	if err != nil {
		return "empty", err
	}

	if len(movies) > 0 {
		saveCache(strconv.Itoa(category), js, cacheDir)
	}

	return string(js[:]), nil
}

func Search(query string, limit int, force int, cacheDir string, cacheDays int64, pages int, tpbHost string, eztvHost string) (string, error) {
	query = strings.TrimSpace(query)
	if force != 1 {
		cache := getCache(query, cacheDir, cacheDays)
		if cache != nil {
			return string(cache[:]), nil
		}
	}

	cancelchan = make(chan bool)
	ctx, cancel = context.WithCancel(context.TODO())
	torrents = make([]tTorrent, 0)

	if tpbHost == "" {
		tpbHost = getTpbHost()
	} else {
		if verbose {
			log.Printf("TPB: Using host %s\n", tpbHost)
		}
	}

	if eztvHost == "" {
		eztvHost = getEztvHost()
	} else {
		if verbose {
			log.Printf("EZTV: Using host %s\n", eztvHost)
		}
	}

	wgt.Add(pages + 1)
	for n := 0; n < pages; n++ {
		go tpbSearch(query, n, tpbHost)
	}
	go eztvSearch(query, eztvHost)
	wgt.Wait()

	if limit > 0 {
		if limit > len(torrents) {
			limit = len(torrents)
		}
		torrents = torrents[0:limit]
	}

	if verbose {
		log.Printf("BUK: Total torrents: %d\n", len(torrents))
	}

	movies = make([]TMovie, 0)

	md := NewTmdb(tmdbApiKey)
	config, err := md.GetConfig()
	if err != nil {
		log.Printf("ERROR: TMDB GetConfig: %s\n", err.Error())
		return "empty", err
	}

	throttle = make(chan int, 10)

	if len(torrents) > 0 {
		for _, torrent := range torrents {
			throttle <- 1
			wg.Add(1)
			if torrent.Category == CategoryTV || torrent.Category == CategoryHDTV {
				go tmdbSearchTv(torrent, config)
			} else {
				go tmdbSearchMovie(torrent, config)
			}
		}
		wg.Wait()
	}

	if verbose {
		log.Printf("BUK: Total movies: %d\n", len(movies))
	}

	sort.Sort(bySeeders(movies))
	js, err := json.MarshalIndent(movies, "", "    ")
	if err != nil {
		return "empty", err
	}

	if len(movies) > 0 {
		saveCache(query, js, cacheDir)
	}

	return string(js[:]), nil
}

func Summary(id int, category int, season int, episode int) (string, error) {
	cancelchan = make(chan bool)
	ctx, cancel = context.WithCancel(context.TODO())
	details = TSummary{}

	wg.Add(1)
	go tmdbSummary(id, category, season, episode)
	wg.Wait()

	if details.Id == 0 {
		return "empty", errors.New("No results")
	}

	js, err := json.MarshalIndent(details, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func Subtitle(movie string, year string, release string, language string, category int, season int, episode int, imdbID string) (string, error) {
	subtitles = make([]TSubtitle, 0)
	language = strings.ToLower(language)

	wgs.Add(3)
	go podnapisi(movie, year, release, language, category, season, episode)
	go opensubtitles(movie, imdbID, year, release, language, category, season, episode)
	go subscene(movie, year, release, language, category, season, episode)
	wgs.Wait()

	if len(subtitles) == 0 && language != "english" {
		if verbose {
			log.Printf("SUB: No %s subtitles, trying with english\n", language)
		}

		wgs.Add(3)
		go podnapisi(movie, year, release, "english", category, season, episode)
		go opensubtitles(movie, imdbID, year, release, "english", category, season, episode)
		go subscene(movie, year, release, "english", category, season, episode)
		wgs.Wait()
	}

	if verbose {
		log.Printf("SUB: Total subtitles: %d\n", len(subtitles))
	}

	sort.Sort(byScore(subtitles))

	js, err := json.MarshalIndent(subtitles, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func UnzipSubtitle(url string, dest string) (string, error) {
	res, err := getResponse(url, true)
	if err != nil {
		log.Printf("ERROR: getResponse %s: %v\n", url, err.Error())
		return "empty", err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("ERROR: ReadAll: %s\n", err.Error())
		return "empty", err
	}
	defer res.Body.Close()

	z, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		log.Printf("ERROR: NewReader: %s\n", err.Error())
		return "empty", err
	}

	exts := []string{".srt", ".ass", ".ssa"}
	if runtime.GOOS != "android" {
		exts = append(exts, ".sub")
	}

	for _, f := range z.File {
		ext := filepath.Ext(f.Name)
		for _, e := range exts {
			if e == ext {
				dst, err := os.Create(filepath.Join(dest, f.Name))
				if err != nil {
					return "empty", err
				}
				defer dst.Close()
				src, err := f.Open()
				if err != nil {
					return "empty", err
				}
				defer src.Close()

				_, err = io.Copy(dst, src)
				if err != nil {
					return "empty", err
				}

				return filepath.Join(dest, f.Name), nil
			}
		}
	}

	return "empty", err
}

func AutoComplete(query string, limit int) (string, error) {
	autocompletes = make([]TItem, 0)
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

func Popular() (string, error) {
	popular = make([]TItem, 0)
	wg.Add(1)
	go tmdbPopular()
	wg.Wait()

	js, err := json.MarshalIndent(popular, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func TopRated() (string, error) {
	toprated = make([]TItem, 0)
	wg.Add(1)
	go tmdbTopRated()
	wg.Wait()

	js, err := json.MarshalIndent(toprated, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func Languages() string {
	langs := make([]string, 0)

	for _, l := range languages {
		langs = append(langs, l.Name)
	}

	return strings.Join(langs, ",")
}

func Genres() (string, error) {
	genres = make([]TGenre, 0)
	wg.Add(1)
	go tmdbGetGenres()
	wg.Wait()

	js, err := json.MarshalIndent(genres, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func Genre(id int, limit int, force int, cacheDir string, cacheDays int64, tpbHost string) (string, error) {
	if force != 1 {
		cache := getCache("genre"+strconv.Itoa(id), cacheDir, cacheDays)
		if cache != nil {
			return string(cache[:]), nil
		}
	}

	cancelchan = make(chan bool)
	ctx, cancel = context.WithCancel(context.TODO())
	bygenre = make([]TMovie, 0)

	if tpbHost == "" {
		tpbHost = getTpbHost()
	} else {
		if verbose {
			log.Printf("TPB: Using host %s\n", tpbHost)
		}
	}

	wg.Add(1)
	go tmdbByGenre(id, limit, tpbHost)
	wg.Wait()

	if verbose {
		log.Printf("BUK: Total movies: %d\n", len(bygenre))
	}

	sort.Sort(bySeeders(bygenre))
	js, err := json.MarshalIndent(bygenre, "", "    ")
	if err != nil {
		return "empty", err
	}

	if len(bygenre) > 0 {
		saveCache("genre"+strconv.Itoa(id), js, cacheDir)
	}

	return string(js[:]), nil
}

func Trailer(videoId string) (string, error) {
	uri, err := vidextr.YouTube(videoId)

	if err != nil {
		return "empty", err
	}

	if uri == "" {
		return "empty", nil
	}

	return uri, nil
}

func Cancel() {
	cancel()
	cancelchan <- true
}

func SetVerbose(v bool) {
	verbose = v
}

func IsValidCategory(category int) bool {
	for _, cat := range Categories {
		if cat == category {
			return true
		}
	}
	return false
}

func TorrentWaitStartup() bool {
	start := time.Now()
	for time.Since(start).Seconds() < 10 {
		s, err := TorrentStatus()
		if err == nil {
			var status TStatus
			err = json.Unmarshal([]byte(s), &status)
			if err == nil && status.State != -1 {
				return true
			}
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func TorrentLargestFile() string {
	tfiles, err := TorrentFiles()
	if err != nil {
		return ""
	}

	var info TLsInfo
	err = json.Unmarshal([]byte(tfiles), &info)
	if err != nil {
		return ""
	}

	if len(info.Files) == 0 {
		return ""
	}

	file := info.Files[0]
	for _, f := range info.Files {
		if f.Size > file.Size {
			file = f
		}
	}

	js, err := json.MarshalIndent(file, "", "    ")
	if err != nil {
		return ""
	}

	return string(js[:])
}

func TorrentStartup(config string) {
	t2http.Startup(config)
}

func TorrentShutdown() {
	t2http.Shutdown()
}

func TorrentStarted() bool {
	return t2http.Started()
}

func TorrentStop() {
	t2http.Stop()
}

func TorrentStatus() (string, error) {
	return t2http.Status()
}

func TorrentFiles() (string, error) {
	return t2http.Ls()
}
