package bukanir

import (
	"encoding/json"
	"fmt"
	"net/url"
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
	Results []video
}

type tmdbGenre struct {
	Id   int
	Name string
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

type video struct {
	Id        string
	Iso_639_1 string
	Key       string
	Name      string
	Site      string
	Size      int
	Type      string
}

func tmdbInit(api_key string) *tmdb {
	return &tmdb{Api_key: api_key}
}

func (t *tmdb) GetConfig() (*tmdbConfig, error) {
	var conf = &tmdbConfig{}
	body, err := httpGetBody(fmt.Sprintf("%s/configuration?api_key=%s",
		tmdbApiUrl, t.Api_key))
	if err != nil {
		return conf, err
	}
	if err := json.Unmarshal(body, &conf); err != nil {
		return conf, err
	}
	return conf, nil
}

func (t *tmdb) SearchMovie(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	body, err := httpGetBody(fmt.Sprintf("%s/search/movie?api_key=%s&query=%s",
		tmdbApiUrl, t.Api_key, url.QueryEscape(query)))
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func (t *tmdb) SearchTv(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	body, err := httpGetBody(fmt.Sprintf("%s/search/tv?api_key=%s&query=%s",
		tmdbApiUrl, t.Api_key, url.QueryEscape(query)))
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func (t *tmdb) AutoCompleteMovie(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	body, err := httpGetBody(fmt.Sprintf("%s/search/movie?search_type=ngram&language=en&api_key=%s&query=%s",
		tmdbApiUrl, t.Api_key, url.QueryEscape(query)))
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func (t *tmdb) AutoCompleteTv(query string) (tmdbResponse, error) {
	var resp tmdbResponse
	body, err := httpGetBody(fmt.Sprintf("%s/search/tv?search_type=ngram&language=en&api_key=%s&query=%s",
		tmdbApiUrl, t.Api_key, url.QueryEscape(query)))
	if err != nil {
		return resp, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func (t *tmdb) GetMovieDetails(id string) (tmdbMovie, error) {
	var movie tmdbMovie
	body, err := httpGetBody(fmt.Sprintf("%s/movie/%s?api_key=%s&append_to_response=credits,videos",
		tmdbApiUrl, id, t.Api_key))
	if err != nil {
		return movie, err
	}
	if err := json.Unmarshal(body, &movie); err != nil {
		return movie, err
	}
	return movie, nil
}

func (t *tmdb) GetTvDetails(id string, season int) (tmdbMovie, error) {
	var movie tmdbMovie
	var uri string
	if season > 0 {
		uri = fmt.Sprintf("%s/tv/%s/season/%d?api_key=%s&append_to_response=credits,videos",
			tmdbApiUrl, id, season, t.Api_key)
	} else {
		uri = fmt.Sprintf("%s/tv/%s?api_key=%s&append_to_response=credits,videos",
			tmdbApiUrl, id, t.Api_key)
	}
	body, err := httpGetBody(uri)
	if err != nil {
		return movie, err
	}
	if err := json.Unmarshal(body, &movie); err != nil {
		return movie, err
	}
	return movie, nil
}

func (t *tmdb) GetTvImages(id string, season int) (tmdbPoster, error) {
	var poster tmdbPoster
	body, err := httpGetBody(fmt.Sprintf("%s/tv/%s/season/%d/images?api_key=%s",
		tmdbApiUrl, id, season, t.Api_key))
	if err != nil {
		return poster, err
	}
	if err := json.Unmarshal(body, &poster); err != nil {
		return poster, err
	}
	return poster, nil
}

func (t *tmdb) GetTvExternals(id string) (tmdbExternals, error) {
	var ext tmdbExternals
	body, err := httpGetBody(fmt.Sprintf("%s/tv/%s/external_ids?api_key=%s",
		tmdbApiUrl, id, t.Api_key))
	if err != nil {
		return ext, err
	}
	if err := json.Unmarshal(body, &ext); err != nil {
		return ext, err
	}
	return ext, nil
}
