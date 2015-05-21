// Copyright 2014, Amahi.  All rights reserved.
// Use of this source code is governed by the
// license that can be found in the LICENSE file.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const base_url string = "http://api.themoviedb.org/3"

type tmdb struct {
	api_key string
	config  *tmdbConfig
}

func tmdbInit(api_key string) *tmdb {
	return &tmdb{api_key: api_key}
}

// response of search
type tmdbResponse struct {
	Page          int
	Results       []tmdbResult
	Total_pages   int
	Total_results int
}

// results format from Tmdb
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

// response of config
type tmdbConfig struct {
	Images imageConfig
}

// Image configurtion
type imageConfig struct {
	Base_url        string
	Secure_base_url string

	//possible sizes for images
	Backdrop_sizes []string
	Logo_sizes     []string
	Poster_sizes   []string
	Profile_sizes  []string
	Still_sizes    []string
}

// Movie metadata structure
type movieMetadata struct {
	Id            int
	Media_type    string
	Backdrop_path string
	Poster_path   string
	Credits       tmdbCredits
	Config        *tmdbConfig
	Imdb_id       string
	Overview      string
	Title         string
	Release_date  string
	Tagline       string
	Runtime       int
	Vote_average  float64
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

type tmdbExternals struct {
	Imdb_id      string
	Freebase_mid string
	Freebase_id  string
	Tvdb_id      int
	Tvrage_id    int
	Id           int
}

func httpGetBody(uri string) ([]byte, error) {
	var err error
	var body []byte
	var res *http.Response

	res, err = http.Get(uri)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 429 {
		sleep, _ := strconv.Atoi(res.Header.Get("Retry-After"))
		time.Sleep(time.Duration(sleep+1) * time.Second)

		res, err = http.Get(uri)
		if err != nil {
			return nil, err
		}
	}

	if res.StatusCode != 200 {
		return nil, error_status(res.StatusCode)
	}

	defer res.Body.Close()
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Search on tmdb for Movies with a given name
func (t *tmdb) SearchMovie(media_name string) (tmdbResponse, error) {
	var resp tmdbResponse
	body, err := httpGetBody(base_url + "/search/movie?api_key=" + t.api_key + "&query=" + url.QueryEscape(media_name))
	if err != nil {
		return tmdbResponse{}, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return tmdbResponse{}, err
	}
	return resp, nil
}

// Search on tmdb for Tv Shows with a given name
func (t *tmdb) SearchTmdbtv(media_name string) (tmdbResponse, error) {
	var resp tmdbResponse
	body, err := httpGetBody(fmt.Sprintf(base_url+"/search/tv?api_key=%s&query=%s", t.api_key, url.QueryEscape(media_name)))
	if err != nil {
		return tmdbResponse{}, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return tmdbResponse{}, err
	}
	return resp, nil
}

// Autocomplete Movies with a given name
func (t *tmdb) AutoCompleteMovie(media_name string) (tmdbResponse, error) {
	var resp tmdbResponse
	body, err := httpGetBody(base_url + "/search/movie?search_type=ngram&language=en&api_key=" + t.api_key + "&query=" + url.QueryEscape(media_name))
	if err != nil {
		return tmdbResponse{}, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return tmdbResponse{}, err
	}
	return resp, nil
}

// Autocomplete Tv Shows with a given name
func (t *tmdb) AutoCompleteTv(media_name string) (tmdbResponse, error) {
	var resp tmdbResponse
	body, err := httpGetBody(base_url + "/search/tv?search_type=ngram&language=en&api_key=" + t.api_key + "&query=" + url.QueryEscape(media_name))
	if err != nil {
		return tmdbResponse{}, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return tmdbResponse{}, err
	}
	return resp, nil
}

// Get configurations from tmdb
func (t *tmdb) GetConfig() (*tmdbConfig, error) {
	if t.config == nil || t.config.Images.Base_url == "" {
		var conf = &tmdbConfig{}
		body, err := httpGetBody(base_url + "/configuration?api_key=" + t.api_key)
		if err != nil {
			return &tmdbConfig{}, err
		}
		if err := json.Unmarshal(body, &conf); err != nil {
			return &tmdbConfig{}, err
		}
		t.config = conf
	}
	return t.config, nil
}

// Get basic information for movie
func (t *tmdb) GetMovieDetails(MediaId string) (movieMetadata, error) {
	var met movieMetadata
	body, err := httpGetBody(base_url + "/movie/" + MediaId + "?api_key=" + t.api_key)
	if err != nil {
		return movieMetadata{}, err
	}
	if err := json.Unmarshal(body, &met); err != nil {
		return movieMetadata{}, err
	}
	return met, nil
}

// Get credits for movie
func (t *tmdb) GetMovieCredits(MediaId string) (tmdbCredits, error) {
	var cred tmdbCredits
	body, err := httpGetBody(base_url + "/movie/" + MediaId + "/credits?api_key=" + t.api_key)
	if err != nil {
		return tmdbCredits{}, err
	}
	if err := json.Unmarshal(body, &cred); err != nil {
		return tmdbCredits{}, err
	}
	return cred, nil
}

// Get basic information for Tv
func (t *tmdb) GetTmdbTvDetails(MediaId string) (movieMetadata, error) {
	var met movieMetadata
	body, err := httpGetBody(fmt.Sprintf(base_url+"/tv/%s?api_key=%s", MediaId, t.api_key))
	if err != nil {
		return movieMetadata{}, err
	}
	if err := json.Unmarshal(body, &met); err != nil {
		return movieMetadata{}, err
	}
	return met, nil
}

// Get credits for Tv
func (t *tmdb) GetTmdbTvCredits(MediaId string, Season int) (tmdbCredits, error) {
	var cred tmdbCredits
	var url string
	if Season > 0 {
		url = fmt.Sprintf(base_url+"/tv/%s/season/%d/credits?api_key=%s", MediaId, Season, t.api_key)
	} else {
		url = fmt.Sprintf(base_url+"/tv/%s/credits?api_key=%s", MediaId, t.api_key)
	}
	body, err := httpGetBody(url)
	if err != nil {
		return tmdbCredits{}, err
	}
	if err := json.Unmarshal(body, &cred); err != nil {
		return tmdbCredits{}, err
	}
	return cred, nil
}

// Get images for Tv season
func (t *tmdb) GetTmdbTvImages(MediaId string, Season int) (tmdbPoster, error) {
	var pos tmdbPoster
	body, err := httpGetBody(fmt.Sprintf(base_url+"/tv/%s/season/%d/images?api_key=%s", MediaId, Season, t.api_key))
	if err != nil {
		return tmdbPoster{}, err
	}
	if err := json.Unmarshal(body, &pos); err != nil {
		return tmdbPoster{}, err
	}
	return pos, nil
}

// Get external ids for Tv
func (t *tmdb) GetTmdbTvExternals(MediaId string) (tmdbExternals, error) {
	var ext tmdbExternals
	body, err := httpGetBody(fmt.Sprintf(base_url+"/tv/%s/external_ids?api_key=%s", MediaId, t.api_key))
	if err != nil {
		return tmdbExternals{}, err
	}
	if err := json.Unmarshal(body, &ext); err != nil {
		return tmdbExternals{}, err
	}
	return ext, nil
}

func error_status(status int) error {
	return errors.New(fmt.Sprintf("Status Code %d received from tmdb", status))
}
