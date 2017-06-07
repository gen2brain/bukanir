// Package bukanir streams movies from bittorrent magnet links
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
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/vidextr"
)

// TMovie type
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

// TSummary type
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

// TSubtitle type
type TSubtitle struct {
	Id           string  `json:"id"`
	Title        string  `json:"title"`
	Year         string  `json:"year"`
	Release      string  `json:"release"`
	DownloadLink string  `json:"downloadLink"`
	Score        float64 `json:"score"`
}

// TItem type
type TItem struct {
	Title string `json:"title"`
	Year  string `json:"year"`
}

// TGenre type
type TGenre struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// TTorrent type
type TTorrent struct {
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

// TConfig type
type TConfig Config

// TStatus type
type TStatus SessionStatus

// TLsInfo type
type TLsInfo LsInfo

// TFileInfo type
type TFileInfo FileStatusInfo

// Sort movies by seeders
type BySeeders []TMovie

func (a BySeeders) Len() int           { return len(a) }
func (a BySeeders) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySeeders) Less(i, j int) bool { return a[i].Seeders > a[j].Seeders }

// Sort subtitles by score
type ByScore []TSubtitle

func (a ByScore) Len() int           { return len(a) }
func (a ByScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByScore) Less(i, j int) bool { return a[i].Score > a[j].Score }

// Sort torrents by seeders
type ByTSeeders []TTorrent

func (a ByTSeeders) Len() int           { return len(a) }
func (a ByTSeeders) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTSeeders) Less(i, j int) bool { return a[i].Seeders > a[j].Seeders }

// Sort movies by season/episodes
type BySeasonEpisode struct {
	movies []TMovie
}

func (a *BySeasonEpisode) Sort(movies []TMovie) {
	a.movies = movies
	sort.Sort(a)
}

func (a *BySeasonEpisode) Len() int {
	return len(a.movies)
}

func (a *BySeasonEpisode) Swap(i, j int) {
	a.movies[i], a.movies[j] = a.movies[j], a.movies[i]
}

func (a *BySeasonEpisode) Less(i, j int) bool {
	p, q := &a.movies[i], &a.movies[j]

	episode := func(c1, c2 *TMovie) bool {
		if c1.Episode == -1 {
			return false
		} else if c2.Episode == -1 {
			return true
		}
		return c1.Episode < c2.Episode
	}

	season := func(c1, c2 *TMovie) bool {
		if c1.Season == -1 {
			return false
		} else if c2.Season == -1 {
			return true
		}
		return c1.Season < c2.Season
	}

	switch {
	case season(p, q):
		return true
	case season(q, p):
		return false
	case episode(p, q):
		return true
	case episode(q, p):
		return false
	}

	return episode(p, q)
}

// Constants
const (
	Version = "2.4"

	tmdbApiUrl = "http://api.themoviedb.org/3"
	tmdbApiKey = "YOUR_API_KEY"

	opensubsUser      = ""
	opensubsPassword  = ""
	opensubsUserAgent = "OSTestUserAgent"
)

// Movies categories
const (
	CategoryMovies   = 201
	CategoryHDmovies = 207
	CategoryTV       = 205
	CategoryHDTV     = 208
)

// Movies categories
var Categories = []int{
	CategoryMovies,
	CategoryHDmovies,
	CategoryTV,
	CategoryHDTV,
}

// TPB onion url
var TpbTor string = "uj3wazyk5u4hnvtk.onion"

// TPB hosts
var TpbHosts = []string{
	"thepiratebay.org",
	"thepiratebay.mk",
	"thepiratebay.lv",
}

// EZTV hosts
var EztvHosts = []string{
	"eztv.ag",
	"eztv.wf",
	"eztv.tf",
	"eztv.yt",
}

// Globals
var (
	movies        []TMovie
	subtitles     []TSubtitle
	autocompletes []TItem
	popular       []TItem
	toprated      []TItem
	genres        []TGenre
	bygenre       []TMovie
	torrents      []TTorrent
	details       TSummary

	wg, wgt, wgs sync.WaitGroup

	verbose    bool
	throttle   chan int
	cancelchan chan bool

	ttor     *tor
	ttorrent *torrent

	ctx, cancel = context.WithCancel(context.TODO())
)

// init starts tor
func init() {
	if runtime.GOOS != "android" {
		ttor = &tor{}

		if !ttor.Exists() {
			log.Printf("ERROR: no tor in $PATH\n")
			return
		}

		tmpDir, err := ioutil.TempDir(os.TempDir(), "tor")
		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
		}

		usr, err := user.Current()
		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
			return
		}

		ttor = NewTor(usr.Name, "9250", "9251", tmpDir)
		ttor.SetDataDir()

		err = ttor.Start()
		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// Category returns movies by category
func Category(category int, limit int, force int, cacheDir string, cacheDays int64, tpbHost string) (string, error) {
	if force != 1 {
		cache := getCache(strconv.Itoa(category), cacheDir, cacheDays)
		if cache != nil {
			return string(cache[:]), nil
		}
	}

	cancelchan = make(chan bool)
	ctx, cancel = context.WithCancel(context.TODO())
	torrents = make([]TTorrent, 0)

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

	throttle = make(chan int, 3*runtime.NumCPU())

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

	sort.Sort(BySeeders(movies))

	js, err := json.MarshalIndent(movies, "", "    ")
	if err != nil {
		return "empty", err
	}

	if len(movies) > 0 {
		saveCache(strconv.Itoa(category), js, cacheDir)
	}

	return string(js[:]), nil
}

// Search returns movies by search query
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
	torrents = make([]TTorrent, 0)

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

	throttle = make(chan int, 3*runtime.NumCPU())

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

	sort.Sort(BySeeders(movies))

	js, err := json.MarshalIndent(movies, "", "    ")
	if err != nil {
		return "empty", err
	}

	if len(movies) > 0 {
		saveCache(query, js, cacheDir)
	}

	return string(js[:]), nil
}

// Summary returns movie summary
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

// Subtitle returns movie subtitles
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

	sort.Sort(ByScore(subtitles))

	js, err := json.MarshalIndent(subtitles, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

// UnzipSubtitle unzip subtitle
func UnzipSubtitle(url string, dest string) (string, error) {
	res, err := getResponse(url)
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

// AutoComplete completes search queries
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

// Popular returns popular movies
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

// TopRated returns top rated movies
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

// Languages returns all supported languages
func Languages() string {
	langs := make([]string, 0)

	for _, l := range languages {
		langs = append(langs, l.Name)
	}

	return strings.Join(langs, ",")
}

// Genres returns all genres
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

// Genre returns movies by genre
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

	sort.Sort(BySeeders(bygenre))
	js, err := json.MarshalIndent(bygenre, "", "    ")
	if err != nil {
		return "empty", err
	}

	if len(bygenre) > 0 {
		saveCache("genre"+strconv.Itoa(id), js, cacheDir)
	}

	return string(js[:]), nil
}

// Trailer returns extracted video url
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

// Cancel cancels context
func Cancel() {
	cancel()
	cancelchan <- true
}

// SetVerbose sets verbosity
func SetVerbose(v bool) {
	verbose = v
}

// IsValidCategory checks if category is valid
func IsValidCategory(category int) bool {
	for _, cat := range Categories {
		if cat == category {
			return true
		}
	}
	return false
}

// TorrentWaitStartup waits for torrent to start
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

// TorrentLargestFile returns largest file from torrent
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

// TorrentStartup starts torrent services
func TorrentStartup(config string) {
	ttorrent = &torrent{}
	ttorrent.Startup(config)
}

// TorrentShutdown shutdowns torrent
func TorrentShutdown() {
	ttorrent.Shutdown()
}

// TorrentRunning checks if torrent is started
func TorrentRunning() bool {
	return ttorrent.Running()
}

// TorrentStop stops torrent
func TorrentStop() {
	ttorrent.Stop()
}

// TorrentStatus returns torrent status
func TorrentStatus() (string, error) {
	return ttorrent.Status()
}

// TorrentFiles returns torrent files
func TorrentFiles() (string, error) {
	return ttorrent.Ls()
}

// TorStart starts tor
func TorStart() error {
	return ttor.Start()
}

// TorStop stops tor
func TorStop() error {
	return ttor.Stop()
}

// TorRunning checks if tor is started
func TorRunning() bool {
	return ttor.Running()
}
