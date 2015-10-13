package bukanir

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gen2brain/vidextr"
)

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
	Quality      string `json:"quality"`
}

type summary struct {
	Id       int      `json:"id"`
	Cast     []string `json:"cast"`
	Genre    []string `json:"genre"`
	Video    string   `json:"video"`
	Director string   `json:"director"`
	Rating   float64  `json:"rating"`
	TagLine  string   `json:"tagline"`
	Overview string   `json:"overview"`
	Runtime  int      `json:"runtime"`
	Imdb_id  string   `json:"imdbId"`
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

type bySeeders []movie

func (a bySeeders) Len() int           { return len(a) }
func (a bySeeders) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a bySeeders) Less(i, j int) bool { return a[i].Seeders > a[j].Seeders }

type byScore []subtitle

func (a byScore) Len() int           { return len(a) }
func (a byScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byScore) Less(i, j int) bool { return a[i].Score > a[j].Score }

const (
	tmdbApiUrl = "http://api.themoviedb.org/3"
	tmdbApiKey = "YOUR_API_KEY"

	opensubsUser      = ""
	opensubsPassword  = ""
	opensubsUserAgent = "OSTestUserAgent"

	podnapisiUrl = "http://podnapisi.net/subtitles/"
	subsceneUrl  = "http://subscene.com"
)

var verbose bool = false

var (
	movies        []movie
	torrents      []torrent
	subtitles     []subtitle
	autocompletes []autocomplete
	details       summary
	wg, wgt, wgs  sync.WaitGroup
)

func tpbTop(category int, host string) {
	defer wgt.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tpbTop: ", r)
		}
	}()

	pb := tpbInit(host)
	results, err := pb.Top(category)
	if err != nil {
		log.Printf("Error making TPB call: %v\n", err.Error())
		return
	}

	torrents = append(torrents, results...)
}

func tpbSearch(query string, page int, host string) {
	defer wgt.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tpbSearch: ", r)
		}
	}()

	pb := tpbInit(host)
	results, err := pb.Search(query, page)
	if err != nil {
		log.Printf("Error making TPB call: %v\n", err.Error())
		return
	}

	torrents = append(torrents, results...)
}

func tmdbSearch(t torrent, config *tmdbConfig) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tmdbSearch: ", r)
		}
	}()

	var err error
	var results tmdbResponse

	md := tmdbInit(tmdbApiKey)

	if t.Category == category_tv || t.Category == category_hdtv {
		results, err = md.SearchTv(t.FormattedTitle)
	} else {
		results, err = md.SearchMovie(t.FormattedTitle)
	}

	if err != nil {
		log.Printf("Error making TMDB call: %v\n", err.Error())
		return
	}

	if results.Total_results == 0 {
		return
	}

	var res *tmdbResult
	if t.Category == category_tv || t.Category == category_hdtv {
		res = &results.Results[0]
	} else {
		res = new(tmdbResult)
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

	var p tmdbPoster
	if t.Category == category_tv || t.Category == category_hdtv {
		p, err = md.GetTvImages(strconv.Itoa(res.Id), t.Season)
		if err != nil {
			log.Printf("Error making TMDB call: %v\n", err.Error())
			return
		}
	}

	if len(config.Images.Poster_sizes) < 5 {
		return
	}

	var posterSmall, posterMedium, posterLarge, posterXLarge string
	if t.Category == category_tv || t.Category == category_hdtv {
		if len(p.Posters) < 1 {
			return
		}
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

	var title string
	var year string
	if t.Category == category_tv || t.Category == category_hdtv {
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
		getQuality(t.Title),
	}
	movies = append(movies, m)
}

func tmdbSummary(id int, category int, season int, episode int) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tmdbSummary: ", r)
		}
	}()

	md := tmdbInit(tmdbApiKey)

	var err error
	var res, res_season tmdbMovie

	if category == category_tv || category == category_hdtv {
		res, err = md.GetTvDetails(strconv.Itoa(id), -1)
		if err != nil {
			log.Printf("Error making TMDB call: %v\n", err.Error())
			return
		}
		res_season, err = md.GetTvDetails(strconv.Itoa(id), season)
		if err != nil {
			log.Printf("Error making TMDB call: %v\n", err.Error())
			return
		}
	} else {
		res, err = md.GetMovieDetails(strconv.Itoa(id))
		if err != nil {
			log.Printf("Error making TMDB call: %v\n", err.Error())
			return
		}
	}

	var overview string = res.Overview
	var director string = getDirector(res.Credits.Crew)
	var video = getVideo(res.Videos.Results)
	var casts []tmdbCast = res.Credits.Cast
	var genres []tmdbGenre = res.Genres
	var imdbId string = strings.Replace(res.Imdb_id, "tt", "", -1)

	if category == category_tv || category == category_hdtv {
		if len(res_season.Episodes) > episode-1 {
			o := res_season.Episodes[episode-1].Overview
			if o != "" {
				overview = o
			}

			if len(res_season.Episodes[episode-1].Crew) > 0 {
				d := getDirector(res_season.Episodes[episode-1].Crew)
				if d != "" {
					director = d
				}
			}
		}

		if len(res_season.Videos.Results) > 0 {
			v := getVideo(res_season.Videos.Results)
			if v != "" {
				video = v
			}
		}

		if len(res_season.Credits.Cast) > 0 {
			casts = res_season.Credits.Cast
		}

		if imdbId == "" {
			ext, err := md.GetTvExternals(strconv.Itoa(id))
			if err != nil {
				log.Printf("Error making TMDB call: %v\n", err.Error())
				return
			}
			imdbId = strings.Replace(ext.Imdb_id, "tt", "", -1)
		}
	}

	details = summary{
		id,
		getCast(casts),
		getGenre(genres),
		video,
		director,
		res.Vote_average,
		res.Tagline,
		overview,
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

	md := tmdbInit(tmdbApiKey)

	movies, err := md.AutoCompleteMovie(query)
	if err != nil {
		log.Printf("Error making TMDB call: %v\n", err.Error())
		return
	}
	tvs, err := md.AutoCompleteTv(query)
	if err != nil {
		log.Printf("Error making TMDB call: %v\n", err.Error())
		return
	}

	if tvs.Total_results+movies.Total_results == 0 {
		return
	}

	for _, movie := range movies.Results {
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

	for _, tv := range tvs.Results {
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

func SetVerbose(v bool) {
	verbose = v
}

func IsValidCategory(category int) bool {
	for _, cat := range categories {
		if cat == category {
			return true
		}
	}
	return false
}

func Category(category int, limit int, force int, cacheDir string, cacheDays int64) (string, error) {
	if force != 1 {
		cache := getCache(strconv.Itoa(category), cacheDir, cacheDays)
		if cache != nil {
			return string(cache[:]), nil
		}
	}

	torrents = make([]torrent, 0)
	host := getHost()

	wgt.Add(1)
	go tpbTop(category, host)
	wgt.Wait()

	if limit > 0 {
		if limit > len(torrents) {
			limit = len(torrents)
		}
		torrents = torrents[0:limit]
	}

	if verbose {
		log.Printf("Total torrents: %d\n", len(torrents))
	}

	movies = make([]movie, 0)

	md := tmdbInit(tmdbApiKey)
	config, err := md.GetConfig()
	if err != nil {
		log.Printf("Error making TMDB call: %v\n", err.Error())
		return "empty", err
	}

	wg.Add(len(torrents))
	for _, torrent := range torrents {
		tmdbSearch(torrent, config)
	}
	wg.Wait()

	if verbose {
		log.Printf("Total movies: %d\n", len(movies))
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

func Search(query string, limit int, force int, cacheDir string, cacheDays int64) (string, error) {
	query = strings.TrimSpace(query)
	if force != 1 {
		cache := getCache(query, cacheDir, cacheDays)
		if cache != nil {
			return string(cache[:]), nil
		}
	}

	torrents = make([]torrent, 0)
	host := getHost()

	wgt.Add(3)
	go tpbSearch(query, 0, host)
	go tpbSearch(query, 1, host)
	go tpbSearch(query, 2, host)
	wgt.Wait()

	if limit > 0 {
		if limit > len(torrents) {
			limit = len(torrents)
		}
		torrents = torrents[0:limit]
	}

	if verbose {
		log.Printf("Total torrents: %d\n", len(torrents))
	}

	movies = make([]movie, 0)

	md := tmdbInit(tmdbApiKey)
	config, err := md.GetConfig()
	if err != nil {
		log.Printf("Error making TMDB call: %v\n", err.Error())
		return "empty", err
	}

	wg.Add(len(torrents))
	for _, torrent := range torrents {
		tmdbSearch(torrent, config)
	}
	wg.Wait()

	if verbose {
		log.Printf("Total movies: %d\n", len(movies))
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
	details = summary{}

	wg.Add(1)
	go tmdbSummary(id, category, season, episode)
	wg.Wait()

	js, err := json.MarshalIndent(details, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func Subtitle(movie string, year string, release string, language string, category int, season int, episode int, imdbID string) (string, error) {
	subtitles = make([]subtitle, 0)
	language = strings.ToLower(language)

	wgs.Add(3)
	go podnapisi(movie, year, release, language, category, season, episode)
	go opensubtitles(movie, imdbID, year, release, language, category, season, episode)
	go subscene(movie, year, release, language, category, season, episode)
	wgs.Wait()

	if len(subtitles) == 0 && language != "english" {
		if verbose {
			log.Printf("Trying with english subtitles\n")
		}

		wgs.Add(3)
		go podnapisi(movie, year, release, "english", category, season, episode)
		go opensubtitles(movie, imdbID, year, release, "english", category, season, episode)
		go subscene(movie, year, release, "english", category, season, episode)
		wgs.Wait()
	}

	if verbose {
		log.Printf("Total subtitles: %d\n", len(subtitles))
	}

	sort.Sort(byScore(subtitles))

	js, err := json.MarshalIndent(subtitles, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func UnzipSubtitle(url string, dest string) (string, error) {
	res, err := httpGetResponse(url)
	if err != nil {
		log.Printf("Error httpGetResponse %s: %v\n", url, err.Error())
		return "empty", err
	}

	if res.StatusCode != 200 {
		return "empty", errors.New(
			fmt.Sprintf("Error UnzipSubtitle: StatusCode %d received", res.StatusCode))
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading body: %v\n", err.Error())
		return "empty", err
	}
	defer res.Body.Close()

	z, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		log.Printf("Error creating zip reader: %v\n", err.Error())
		return "empty", err
	}

	for _, f := range z.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".srt") ||
			strings.HasSuffix(strings.ToLower(f.Name), ".ass") ||
			strings.HasSuffix(strings.ToLower(f.Name), ".ssa") {
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

	return "empty", err
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
