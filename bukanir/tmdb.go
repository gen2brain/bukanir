package bukanir

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
)

type tmdb struct {
	Api_key string
}

type tmdbConfig struct {
	Images tmdbImageConfig
}

type tmdbResponse struct {
	Page          int
	Results       []tmdbResult
	Total_pages   int
	Total_results int
}

type tmdbResult struct {
	Adult          bool
	Name           string
	Backdrop_path  string
	Id             int
	Original_name  string
	Original_title string
	First_air_date string
	Release_date   string
	Poster_path    string
	Title          string
	Media_type     string
	Profile_path   string
}

type tmdbImageConfig struct {
	Base_url        string
	Secure_base_url string
	Backdrop_sizes  []string
	Logo_sizes      []string
	Poster_sizes    []string
	Profile_sizes   []string
	Still_sizes     []string
}

type tmdbMovie struct {
	Id            int
	Media_type    string
	Backdrop_path string
	Poster_path   string
	Episodes      []tmdbEpisode
	Credits       tmdbCredits
	Config        *tmdbConfig
	Videos        tmdbVideos
	Genres        []tmdbGenre
	Imdb_id       string
	Overview      string
	Title         string
	Release_date  string
	Tagline       string
	Runtime       int
	Vote_average  float64
}

type tmdbEpisode struct {
	Air_date       string
	Season_number  int
	Episode_number int
	Overview       string
	Crew           []tmdbCrew
}

type tmdbCredits struct {
	Id   int
	Cast []tmdbCast
	Crew []tmdbCrew
}

type tmdbCast struct {
	Character    string
	Name         string
	Profile_path string
}

type tmdbCrew struct {
	Department   string
	Name         string
	Job          string
	Profile_path string
}

type tmdbPoster struct {
	Id      int
	Posters []poster
}

type tmdbVideos struct {
	Id      int
	Results []tmdbVideo
}

type tmdbGenre struct {
	Id   int
	Name string
}

type tmdbGenres struct {
	Genres []tmdbGenre
}

type tmdbExternals struct {
	Imdb_id      string
	Freebase_mid string
	Freebase_id  string
	Tvdb_id      int
	Tvrage_id    int
	Id           int
}

type poster struct {
	Aspect_ratio float64
	File_path    string
	Height       int
	Id           string
	Iso_639_1    string
	Vote_average float64
	Vote_count   int
	Width        int
}

type tmdbVideo struct {
	Id        string
	Iso_639_1 string
	Key       string
	Name      string
	Site      string
	Size      int
	Type      string
}

// Returns new tmdb
func NewTmdb(api_key string) *tmdb {
	return &tmdb{Api_key: api_key}
}

// Returns new tmdb config
func (t *tmdb) GetConfig() (*tmdbConfig, error) {
	var conf = &tmdbConfig{}
	uri := fmt.Sprintf("%s/configuration?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s", t.Api_key), "", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return conf, err
	}
	if err := json.Unmarshal(body, &conf); err != nil {
		return conf, err
	}
	return conf, nil
}

// Returns movie search response
func (t *tmdb) SearchMovie(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s", tmdbApiUrl, t.Api_key, url.QueryEscape(query))

	if verbose {
		//log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s&", t.Api_key), "?", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Returns tv show search response
func (t *tmdb) SearchTv(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s", tmdbApiUrl, t.Api_key, url.QueryEscape(query))

	if verbose {
		//log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s&", t.Api_key), "?", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Returns movie autocomplete response
func (t *tmdb) AutoCompleteMovie(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/search/movie?api_key=%s&search_type=ngram&language=en&query=%s", tmdbApiUrl, t.Api_key, url.QueryEscape(query))

	if verbose {
		//log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s&", t.Api_key), "?", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Returns tv show autocomplete response
func (t *tmdb) AutoCompleteTv(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/search/tv?api_key=%s&search_type=ngram&language=en&query=%s", tmdbApiUrl, t.Api_key, url.QueryEscape(query))

	if verbose {
		//log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s&", t.Api_key), "?", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Returns movie details
func (t *tmdb) GetMovieDetails(id string) (tmdbMovie, error) {
	var movie tmdbMovie
	uri := fmt.Sprintf("%s/movie/%s?api_key=%s&append_to_response=credits,videos", tmdbApiUrl, id, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s&", t.Api_key), "?", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return movie, err
	}
	if err := json.Unmarshal(body, &movie); err != nil {
		return movie, err
	}
	return movie, nil
}

// Returns tv show details
func (t *tmdb) GetTvDetails(id string, season int) (tmdbMovie, error) {
	var movie tmdbMovie
	var uri string
	if season > 0 {
		uri = fmt.Sprintf("%s/tv/%s/season/%d?api_key=%s&append_to_response=credits,videos", tmdbApiUrl, id, season, t.Api_key)
	} else {
		uri = fmt.Sprintf("%s/tv/%s?api_key=%s&append_to_response=credits,videos", tmdbApiUrl, id, t.Api_key)
	}

	if verbose {
		log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s&", t.Api_key), "?", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return movie, err
	}
	if err := json.Unmarshal(body, &movie); err != nil {
		return movie, err
	}
	return movie, nil
}

// Returns tv show images
func (t *tmdb) GetTvImages(id string, season int) (tmdbPoster, error) {
	var poster tmdbPoster
	uri := fmt.Sprintf("%s/tv/%s/season/%d/images?api_key=%s", tmdbApiUrl, id, season, t.Api_key)

	if verbose {
		//log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s", t.Api_key), "", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return poster, err
	}
	if err := json.Unmarshal(body, &poster); err != nil {
		return poster, err
	}
	return poster, nil
}

// Returns tv show externals
func (t *tmdb) GetTvExternals(id string) (tmdbExternals, error) {
	var ext tmdbExternals
	uri := fmt.Sprintf("%s/tv/%s/external_ids?api_key=%s", tmdbApiUrl, id, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s", t.Api_key), "", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return ext, err
	}
	if err := json.Unmarshal(body, &ext); err != nil {
		return ext, err
	}
	return ext, nil
}

// Returns popular movies response
func (t *tmdb) PopularMovies() (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/movie/popular?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s", t.Api_key), "", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Returns popular tv shows response
func (t *tmdb) PopularTv() (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/tv/popular?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s", t.Api_key), "", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Returns top rated movies response
func (t *tmdb) TopRatedMovies() (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/movie/top_rated?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s", t.Api_key), "", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Returns top rated tv shows response
func (t *tmdb) TopRatedTv() (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/tv/top_rated?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s", t.Api_key), "", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Returns genres
func (t *tmdb) Genres() (tmdbGenres, error) {
	var resp tmdbGenres
	uri := fmt.Sprintf("%s/genre/movie/list?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s", t.Api_key), "", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Returns movies for genre response
func (t *tmdb) Genre(id, page int) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/discover/movie?api_key=%s&sort_by=popularity.desc&with_genres=%d&page=%d", tmdbApiUrl, t.Api_key, id, page)

	if verbose {
		log.Printf("TMDB: GET %s\n", strings.Replace(uri, fmt.Sprintf("?api_key=%s&", t.Api_key), "?", -1))
	}

	body, err := getBody(uri)
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}
