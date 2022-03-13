package bukanir

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
)

// tmdb type
type tmdb struct {
	Api_key string
}

// tmdbConfig type
type tmdbConfig struct {
	Images tmdbImageConfig
}

// tmdbResponse type
type tmdbResponse struct {
	Page          int
	Results       []tmdbResult
	Total_pages   int
	Total_results int
}

// tmdbResult type
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

// tmdbImageConfig type
type tmdbImageConfig struct {
	Base_url        string
	Secure_base_url string
	Backdrop_sizes  []string
	Logo_sizes      []string
	Poster_sizes    []string
	Profile_sizes   []string
	Still_sizes     []string
}

// tmdbMovie type
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

// tmdbEpisode type
type tmdbEpisode struct {
	Air_date       string
	Season_number  int
	Episode_number int
	Overview       string
	Crew           []tmdbCrew
}

// tmdbCredits type
type tmdbCredits struct {
	Id   int
	Cast []tmdbCast
	Crew []tmdbCrew
}

// tmdbCast type
type tmdbCast struct {
	Id           int
	Character    string
	Name         string
	Profile_path string
}

// tmdbCrew type
type tmdbCrew struct {
	Id           int
	Department   string
	Name         string
	Job          string
	Profile_path string
}

// tmdbPoster type
type tmdbPoster struct {
	Id      int
	Posters []poster
}

// tmdbVideos type
type tmdbVideos struct {
	Id      int
	Results []tmdbVideo
}

// tmdbGenre type
type tmdbGenre struct {
	Id   int
	Name string
}

// tmdbGenres type
type tmdbGenres struct {
	Genres []tmdbGenre
}

// tmdbExternals type
type tmdbExternals struct {
	Imdb_id      string
	Freebase_mid string
	Freebase_id  string
	Tvdb_id      int
	Tvrage_id    int
	Id           int
}

// tmdbVideo type
type tmdbVideo struct {
	Id        string
	Iso_639_1 string
	Key       string
	Name      string
	Site      string
	Size      int
	Type      string
}

// poster type
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

// NewTmdb returns new tmdb
func NewTmdb(api_key string) *tmdb {
	return &tmdb{Api_key: api_key}
}

// GetConfig returns new tmdb config
func (t *tmdb) GetConfig() (*tmdbConfig, error) {
	var conf = &tmdbConfig{}
	uri := fmt.Sprintf("%s/configuration?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, false))
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

// SearchMovie returns movie search response
func (t *tmdb) SearchMovie(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s", tmdbApiUrl, t.Api_key, url.QueryEscape(query))

	if verbose {
		//log.Printf("TMDB: GET %s\n", t.safeUri(uri, true))
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

// SearchTv returns tv show search response
func (t *tmdb) SearchTv(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s", tmdbApiUrl, t.Api_key, url.QueryEscape(query))

	if verbose {
		//log.Printf("TMDB: GET %s\n", t.safeUri(uri, true))
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

// AutoCompleteMovie returns movie autocomplete response
func (t *tmdb) AutoCompleteMovie(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/search/movie?api_key=%s&search_type=ngram&language=en&query=%s", tmdbApiUrl, t.Api_key, url.QueryEscape(query))

	if verbose {
		//log.Printf("TMDB: GET %s\n", t.safeUri(uri, true))
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

// AutoCompleteTv returns tv show autocomplete response
func (t *tmdb) AutoCompleteTv(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/search/tv?api_key=%s&search_type=ngram&language=en&query=%s", tmdbApiUrl, t.Api_key, url.QueryEscape(query))

	if verbose {
		//log.Printf("TMDB: GET %s\n", t.safeUri(uri, true))
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

// GetMovieDetails returns movie details
func (t *tmdb) GetMovieDetails(id string) (tmdbMovie, error) {
	var movie tmdbMovie
	uri := fmt.Sprintf("%s/movie/%s?api_key=%s&append_to_response=credits,videos", tmdbApiUrl, id, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, true))
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

// GetTvDetails returns tv show details
func (t *tmdb) GetTvDetails(id string, season int) (tmdbMovie, error) {
	var movie tmdbMovie
	var uri string
	if season > 0 {
		uri = fmt.Sprintf("%s/tv/%s/season/%d?api_key=%s&append_to_response=credits,videos", tmdbApiUrl, id, season, t.Api_key)
	} else {
		uri = fmt.Sprintf("%s/tv/%s?api_key=%s&append_to_response=credits,videos", tmdbApiUrl, id, t.Api_key)
	}

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, true))
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

// GetTvImages returns tv show images
func (t *tmdb) GetTvImages(id string, season int) (tmdbPoster, error) {
	var poster tmdbPoster
	uri := fmt.Sprintf("%s/tv/%s/season/%d/images?api_key=%s", tmdbApiUrl, id, season, t.Api_key)

	if verbose {
		//log.Printf("TMDB: GET %s\n", t.safeUri(uri, false))
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

// GetTvExternals returns tv show externals
func (t *tmdb) GetTvExternals(id string) (tmdbExternals, error) {
	var ext tmdbExternals
	uri := fmt.Sprintf("%s/tv/%s/external_ids?api_key=%s", tmdbApiUrl, id, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, false))
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

// PopularMovies returns popular movies response
func (t *tmdb) PopularMovies() (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/movie/popular?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, false))
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

// PopularTv returns popular tv shows response
func (t *tmdb) PopularTv() (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/tv/popular?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, false))
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

// TopRatedMovies returns top rated movies response
func (t *tmdb) TopRatedMovies() (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/movie/top_rated?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, false))
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

// TopRatedTv returns top rated tv shows response
func (t *tmdb) TopRatedTv() (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/tv/top_rated?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, false))
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

// Genres returns available genres
func (t *tmdb) Genres() (tmdbGenres, error) {
	var resp tmdbGenres
	uri := fmt.Sprintf("%s/genre/movie/list?api_key=%s", tmdbApiUrl, t.Api_key)

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, false))
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

// Genre returns movies for genre response
func (t *tmdb) Genre(id, page int) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/discover/movie?api_key=%s&sort_by=popularity.desc&with_genres=%d&page=%d", tmdbApiUrl, t.Api_key, id, page)

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, true))
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

// MoviesWithCast returns movies for cast response
func (t *tmdb) MoviesWithCast(id, page int) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/discover/movie?api_key=%s&sort_by=popularity.desc&with_cast=%d&page=%d", tmdbApiUrl, t.Api_key, id, page)

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, true))
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

// MoviesWithCrew returns movies for crew response
func (t *tmdb) MoviesWithCrew(id, page int) (tmdbResponse, error) {
	var resp tmdbResponse
	uri := fmt.Sprintf("%s/discover/movie?api_key=%s&sort_by=popularity.desc&with_crew=%d&page=%d", tmdbApiUrl, t.Api_key, id, page)

	if verbose {
		log.Printf("TMDB: GET %s\n", t.safeUri(uri, true))
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

func (t *tmdb) safeUri(uri string, b bool) string {
	if b {
		return strings.Replace(uri, fmt.Sprintf("?api_key=%s&", t.Api_key), "?", -1)
	} else {
		return strings.Replace(uri, fmt.Sprintf("?api_key=%s", t.Api_key), "", -1)
	}
}
