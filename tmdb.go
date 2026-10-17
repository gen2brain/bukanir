package bukanir

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// tmdbBaseURL is the TMDB v3 API root.
const tmdbBaseURL = "https://api.themoviedb.org/3"

// Category is a torrent content category, matching the source's numeric codes.
type Category int

const (
	CategoryMovies   Category = 201
	CategoryHDMovies Category = 207
	CategoryTV       Category = 205
	CategoryHDTV     Category = 208
)

// Categories lists the supported categories.
var Categories = []Category{CategoryMovies, CategoryHDMovies, CategoryTV, CategoryHDTV}

// Valid reports whether c is a supported category.
func (c Category) Valid() bool {
	switch c {
	case CategoryMovies, CategoryHDMovies, CategoryTV, CategoryHDTV:
		return true
	}
	return false
}

// IsTV reports whether c is a television category.
func (c Category) IsTV() bool {
	return c == CategoryTV || c == CategoryHDTV
}

// TMovie is a search result: a torrent paired with its TMDB metadata.
type TMovie struct {
	ID           int      `json:"id"`
	Title        string   `json:"title"`
	Year         int      `json:"year"`
	PosterSmall  string   `json:"posterSmall"`
	PosterMedium string   `json:"posterMedium"`
	PosterLarge  string   `json:"posterLarge"`
	PosterXLarge string   `json:"posterXLarge"`
	Size         int64    `json:"size"`
	SizeHuman    string   `json:"sizeHuman"`
	Seeders      int      `json:"seeders"`
	Magnet       string   `json:"magnet"`
	Release      string   `json:"release"`
	Category     Category `json:"category"`
	Season       int      `json:"season"`
	Episode      int      `json:"episode"`
	Quality      string   `json:"quality"`
}

// TSummary is the detailed metadata for a movie or episode.
type TSummary struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	Year       int      `json:"year"`
	Overview   string   `json:"overview"`
	Tagline    string   `json:"tagline"`
	Rating     float64  `json:"rating"`
	Runtime    int      `json:"runtime"`
	Cast       []string `json:"cast"`
	CastIDs    []int    `json:"castIds"`
	Genres     []string `json:"genres"`
	Director   string   `json:"director"`
	DirectorID int      `json:"directorId"`
	TrailerKey string   `json:"trailerKey"` // YouTube video key
	IMDbID     string   `json:"imdbId"`
}

// TItem is a lightweight title suggestion.
type TItem struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	Media string `json:"media,omitempty"`
}

// TGenre is a TMDB genre.
type TGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type tmdbSearchResponse struct {
	Page         int          `json:"page"`
	Results      []tmdbResult `json:"results"`
	TotalPages   int          `json:"total_pages"`
	TotalResults int          `json:"total_results"`
}

// tmdbResult is one entry in a search/discover/list response.
type tmdbResult struct {
	ID            int     `json:"id"`
	MediaType     string  `json:"media_type"`
	Title         string  `json:"title"`
	Name          string  `json:"name"`
	OriginalTitle string  `json:"original_title"`
	OriginalName  string  `json:"original_name"`
	ReleaseDate   string  `json:"release_date"`
	FirstAirDate  string  `json:"first_air_date"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	ProfilePath   string  `json:"profile_path"`
	Overview      string  `json:"overview"`
	VoteAverage   float64 `json:"vote_average"`
}

func (r tmdbResult) title() string {
	if r.Title != "" {
		return r.Title
	}
	return r.Name
}

func (r tmdbResult) year() int {
	return parseYear(r.ReleaseDate, r.FirstAirDate)
}

type tmdbConfiguration struct {
	Images tmdbImageConfig `json:"images"`
}

type tmdbImageConfig struct {
	BaseURL       string   `json:"base_url"`
	SecureBaseURL string   `json:"secure_base_url"`
	PosterSizes   []string `json:"poster_sizes"`
	BackdropSizes []string `json:"backdrop_sizes"`
	ProfileSizes  []string `json:"profile_sizes"`
}

type tmdbDetails struct {
	ID           int           `json:"id"`
	ImdbID       string        `json:"imdb_id"`
	Title        string        `json:"title"`
	Name         string        `json:"name"`
	Overview     string        `json:"overview"`
	Tagline      string        `json:"tagline"`
	ReleaseDate  string        `json:"release_date"`
	FirstAirDate string        `json:"first_air_date"`
	Runtime      int           `json:"runtime"`
	VoteAverage  float64       `json:"vote_average"`
	PosterPath   string        `json:"poster_path"`
	Genres       []tmdbGenre   `json:"genres"`
	Credits      tmdbCredits   `json:"credits"`
	Videos       tmdbVideos    `json:"videos"`
	Episodes     []tmdbEpisode `json:"episodes"` // present on TV season details
}

type tmdbGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type tmdbCredits struct {
	Cast []tmdbPerson `json:"cast"`
	Crew []tmdbPerson `json:"crew"`
}

type tmdbPerson struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Character  string `json:"character"`
	Department string `json:"department"`
	Job        string `json:"job"`
}

type tmdbVideos struct {
	Results []tmdbVideo `json:"results"`
}

type tmdbVideo struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Site string `json:"site"`
	Type string `json:"type"`
}

type tmdbEpisode struct {
	AirDate       string       `json:"air_date"`
	SeasonNumber  int          `json:"season_number"`
	EpisodeNumber int          `json:"episode_number"`
	Overview      string       `json:"overview"`
	Crew          []tmdbPerson `json:"crew"`
}

type tmdbImages struct {
	Posters []tmdbImage `json:"posters"`
}

type tmdbImage struct {
	FilePath string `json:"file_path"`
}

type tmdbExternalIDs struct {
	ImdbID string `json:"imdb_id"`
}

type tmdbGenreList struct {
	Genres []tmdbGenre `json:"genres"`
}

func (c *Client) tmdbURL(path string, q url.Values) string {
	if q == nil {
		q = url.Values{}
	}
	q.Set("api_key", c.cfg.tmdbKey())
	return c.tmdbBase + path + "?" + q.Encode()
}

// tmdbConfiguration fetches the image configuration once and memoizes it.
func (c *Client) tmdbConfiguration(ctx context.Context) (*tmdbConfiguration, error) {
	c.tmdbMu.Lock()
	defer c.tmdbMu.Unlock()
	if c.tmdbConf != nil {
		return c.tmdbConf, nil
	}
	var conf tmdbConfiguration
	if err := c.getJSON(ctx, c.tmdbURL("/configuration", nil), &conf); err != nil {
		return nil, err
	}
	c.tmdbConf = &conf
	return &conf, nil
}

func (c *Client) tmdbSearchMovie(ctx context.Context, query string) (*tmdbSearchResponse, error) {
	return c.tmdbSearch(ctx, "/search/movie", query)
}

func (c *Client) tmdbSearchTV(ctx context.Context, query string) (*tmdbSearchResponse, error) {
	return c.tmdbSearch(ctx, "/search/tv", query)
}

func (c *Client) tmdbSearch(ctx context.Context, path, query string) (*tmdbSearchResponse, error) {
	var out tmdbSearchResponse
	err := c.getJSON(ctx, c.tmdbURL(path, url.Values{"query": {query}}), &out)
	return &out, err
}

func (c *Client) tmdbMovieDetails(ctx context.Context, id int) (*tmdbDetails, error) {
	q := url.Values{"append_to_response": {"credits,videos"}}
	var out tmdbDetails
	err := c.getJSON(ctx, c.tmdbURL("/movie/"+strconv.Itoa(id), q), &out)
	return &out, err
}

// tmdbTVDetails fetches series details, or season details when season > 0.
func (c *Client) tmdbTVDetails(ctx context.Context, id, season int) (*tmdbDetails, error) {
	path := "/tv/" + strconv.Itoa(id)
	if season > 0 {
		path += "/season/" + strconv.Itoa(season)
	}
	q := url.Values{"append_to_response": {"credits,videos"}}
	var out tmdbDetails
	err := c.getJSON(ctx, c.tmdbURL(path, q), &out)
	return &out, err
}

func (c *Client) tmdbTVImages(ctx context.Context, id, season int) (*tmdbImages, error) {
	path := fmt.Sprintf("/tv/%d/season/%d/images", id, season)
	var out tmdbImages
	err := c.getJSON(ctx, c.tmdbURL(path, nil), &out)
	return &out, err
}

func (c *Client) tmdbTVExternalIDs(ctx context.Context, id int) (*tmdbExternalIDs, error) {
	var out tmdbExternalIDs
	err := c.getJSON(ctx, c.tmdbURL(fmt.Sprintf("/tv/%d/external_ids", id), nil), &out)
	return &out, err
}

// tmdbList fetches a paged list such as movie/popular or tv/top_rated.
func (c *Client) tmdbList(ctx context.Context, media, kind string) (*tmdbSearchResponse, error) {
	var out tmdbSearchResponse
	err := c.getJSON(ctx, c.tmdbURL("/"+media+"/"+kind, nil), &out)
	return &out, err
}

func (c *Client) tmdbGenres(ctx context.Context) ([]tmdbGenre, error) {
	var out tmdbGenreList
	if err := c.getJSON(ctx, c.tmdbURL("/genre/movie/list", nil), &out); err != nil {
		return nil, err
	}
	return out.Genres, nil
}

// tmdbDiscoverMovie runs a discover query filtered by with equal to id.
func (c *Client) tmdbDiscoverMovie(ctx context.Context, with string, id, page int) (*tmdbSearchResponse, error) {
	q := url.Values{
		"sort_by": {"popularity.desc"},
		with:      {strconv.Itoa(id)},
		"page":    {strconv.Itoa(page)},
	}
	var out tmdbSearchResponse
	err := c.getJSON(ctx, c.tmdbURL("/discover/movie", q), &out)
	return &out, err
}

// posterURL builds a poster URL for path at the given size index.
func (conf *tmdbConfiguration) posterURL(path string, sizeIndex int) string {
	if path == "" {
		return ""
	}
	sizes := conf.Images.PosterSizes
	base := conf.Images.SecureBaseURL
	if base == "" {
		base = conf.Images.BaseURL
	}
	if len(sizes) == 0 || base == "" {
		return ""
	}
	if sizeIndex < 0 || sizeIndex >= len(sizes) {
		sizeIndex = len(sizes) - 1
	}
	return base + sizes[sizeIndex] + path
}

// parseYear returns the year from the first non-empty "YYYY-MM-DD" date.
func parseYear(dates ...string) int {
	for _, d := range dates {
		if len(d) >= 4 {
			if y, err := strconv.Atoi(d[:4]); err == nil {
				return y
			}
		}
	}
	return 0
}

// trailerKey returns the YouTube key of the first trailer.
func (v tmdbVideos) trailerKey() string {
	var fallback string
	for _, vid := range v.Results {
		if !strings.EqualFold(vid.Site, "YouTube") {
			continue
		}
		if strings.EqualFold(vid.Type, "Trailer") {
			return vid.Key
		}
		if fallback == "" {
			fallback = vid.Key
		}
	}
	return fallback
}

func (cr tmdbCredits) director() (name string, id int) {
	return directorOf(cr.Crew)
}

func directorOf(crew []tmdbPerson) (name string, id int) {
	for _, p := range crew {
		if strings.EqualFold(p.Job, "Director") {
			return p.Name, p.ID
		}
	}
	return "", 0
}

func castNames(cast []tmdbPerson) []string {
	names := make([]string, 0, len(cast))
	for _, p := range cast {
		names = append(names, p.Name)
	}
	return names
}

func castIDs(cast []tmdbPerson) []int {
	ids := make([]int, 0, len(cast))
	for _, p := range cast {
		ids = append(ids, p.ID)
	}
	return ids
}

func genreNames(genres []tmdbGenre) []string {
	names := make([]string, 0, len(genres))
	for _, g := range genres {
		names = append(names, g.Name)
	}
	return names
}

const discoverPages = 5

// Category returns the top torrents for a category, sorted by seeders.
func (c *Client) Category(ctx context.Context, category Category) ([]TMovie, error) {
	if !category.Valid() {
		return nil, fmt.Errorf("bukanir: invalid category %d", category)
	}

	cacheKey := c.cache.key("category", strconv.Itoa(int(category)))
	if data, ok := c.cache.get(cacheKey); ok {
		var cached []TMovie
		if json.Unmarshal(data, &cached) == nil {
			return cached, nil
		}
	}

	torrents, err := c.tpbTop(ctx, category)
	if err != nil {
		return nil, err
	}
	conf, err := c.tmdbConfiguration(ctx)
	if err != nil {
		return nil, err
	}
	movies := boundedMap(ctx, torrents, enrichConcurrency, func(ctx context.Context, tr torrentResult) (TMovie, bool) {
		return c.enrich(ctx, conf, tr)
	})
	sortMovies(movies, "seeders")

	if data, err := json.Marshal(movies); err == nil {
		_ = c.cache.put(cacheKey, data)
	}
	return movies, nil
}

// Genre returns popular movies in a TMDB genre, each paired with a torrent.
func (c *Client) Genre(ctx context.Context, genreID, limit int) ([]TMovie, error) {
	return c.discover(ctx, "with_genres", genreID, limit)
}

// Cast returns popular movies featuring a TMDB person (as cast).
func (c *Client) Cast(ctx context.Context, personID, limit int) ([]TMovie, error) {
	return c.discover(ctx, "with_cast", personID, limit)
}

// Crew returns popular movies involving a TMDB person (as crew, e.g. director).
func (c *Client) Crew(ctx context.Context, personID, limit int) ([]TMovie, error) {
	return c.discover(ctx, "with_crew", personID, limit)
}

// discover browses TMDB movies by a filter, pairing each with a torrent.
func (c *Client) discover(ctx context.Context, with string, id, limit int) ([]TMovie, error) {
	cacheKey := c.cache.key("discover", with, strconv.Itoa(id))
	if data, ok := c.cache.get(cacheKey); ok {
		var cached []TMovie
		if json.Unmarshal(data, &cached) == nil {
			return limitMovies(cached, limit), nil
		}
	}

	var results []tmdbResult
	for page := 1; page <= discoverPages; page++ {
		resp, err := c.tmdbDiscoverMovie(ctx, with, id, page)
		if err != nil {
			break
		}
		results = append(results, resp.Results...)
		if page >= resp.TotalPages {
			break
		}
	}
	if len(results) == 0 {
		return nil, nil
	}
	conf, err := c.tmdbConfiguration(ctx)
	if err != nil {
		return nil, err
	}

	movies := boundedMap(ctx, results, enrichConcurrency, func(ctx context.Context, r tmdbResult) (TMovie, bool) {
		return c.torrentForResult(ctx, conf, r)
	})
	sortMovies(movies, "seeders")

	if data, err := json.Marshal(movies); err == nil {
		_ = c.cache.put(cacheKey, data)
	}
	return limitMovies(movies, limit), nil
}

// torrentForResult finds the best torrent for a TMDB movie result.
func (c *Client) torrentForResult(ctx context.Context, conf *tmdbConfiguration, r tmdbResult) (TMovie, bool) {
	torrents, err := c.tpbSearch(ctx, r.title(), "movies")
	if err != nil || len(torrents) == 0 {
		return TMovie{}, false
	}
	sort.SliceStable(torrents, func(i, j int) bool { return torrents[i].Seeders > torrents[j].Seeders })

	ry := r.year()
	for _, tr := range torrents {
		if ry != 0 && tr.Year != 0 && abs(tr.Year-ry) > 1 {
			continue
		}
		return buildMovie(conf, r, r.PosterPath, tr), true
	}
	return TMovie{}, false
}

func limitMovies(movies []TMovie, limit int) []TMovie {
	if limit > 0 && len(movies) > limit {
		return movies[:limit]
	}
	return movies
}

// Popular returns popular movies followed by popular TV shows.
func (c *Client) Popular(ctx context.Context) ([]TItem, error) {
	return c.movieAndTVList(ctx, "popular")
}

// TopRated returns top-rated movies followed by top-rated TV shows.
func (c *Client) TopRated(ctx context.Context) ([]TItem, error) {
	return c.movieAndTVList(ctx, "top_rated")
}

func (c *Client) movieAndTVList(ctx context.Context, kind string) ([]TItem, error) {
	movies, err := c.tmdbList(ctx, "movie", kind)
	if err != nil {
		return nil, err
	}
	tv, err := c.tmdbList(ctx, "tv", kind)
	if err != nil {
		return nil, err
	}
	return append(taggedItems(movies.Results, "movie"), taggedItems(tv.Results, "tv")...), nil
}

func taggedItems(results []tmdbResult, media string) []TItem {
	items := itemsFromResults(results)
	for i := range items {
		items[i].Media = media
	}
	return items
}

// Genres returns the list of movie genres.
func (c *Client) Genres(ctx context.Context) ([]TGenre, error) {
	genres, err := c.tmdbGenres(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]TGenre, len(genres))
	for i, g := range genres {
		out[i] = TGenre(g)
	}
	return out, nil
}

// Autocomplete returns title suggestions for a partial query.
func (c *Client) Autocomplete(ctx context.Context, query string) ([]TItem, error) {
	movies, err := c.tmdbSearchMovie(ctx, query)
	if err != nil {
		return nil, err
	}
	tv, err := c.tmdbSearchTV(ctx, query)
	if err != nil {
		return nil, err
	}

	results := append(movies.Results, tv.Results...)
	items := make([]TItem, 0, len(results))
	for _, r := range results {
		if r.title() == "" || r.year() == 0 {
			continue
		}
		items = append(items, TItem{Title: r.title(), Year: r.year()})
	}
	return items, nil
}

// Summary returns detailed metadata for a movie or TV episode.
func (c *Client) Summary(ctx context.Context, id int, category Category, season, episode int) (TSummary, error) {
	if category.IsTV() {
		return c.tvSummary(ctx, id, season, episode)
	}
	return c.movieSummary(ctx, id)
}

func (c *Client) movieSummary(ctx context.Context, id int) (TSummary, error) {
	d, err := c.tmdbMovieDetails(ctx, id)
	if err != nil {
		return TSummary{}, err
	}
	return summaryFromDetails(d), nil
}

func (c *Client) tvSummary(ctx context.Context, id, season, episode int) (TSummary, error) {
	series, err := c.tmdbTVDetails(ctx, id, 0)
	if err != nil {
		return TSummary{}, err
	}
	s := summaryFromDetails(series)

	if ext, err := c.tmdbTVExternalIDs(ctx, id); err == nil && ext.ImdbID != "" {
		s.IMDbID = ext.ImdbID
	}

	if season > 0 {
		if sd, err := c.tmdbTVDetails(ctx, id, season); err == nil {
			for _, ep := range sd.Episodes {
				if ep.EpisodeNumber != episode {
					continue
				}
				if ep.Overview != "" {
					s.Overview = ep.Overview
				}
				if name, dirID := directorOf(ep.Crew); name != "" {
					s.Director, s.DirectorID = name, dirID
				}
				break
			}
		}
	}
	return s, nil
}

func summaryFromDetails(d *tmdbDetails) TSummary {
	name, dirID := d.Credits.director()
	title := d.Title
	if title == "" {
		title = d.Name
	}
	return TSummary{
		ID:         d.ID,
		Title:      title,
		Year:       parseYear(d.ReleaseDate, d.FirstAirDate),
		Overview:   d.Overview,
		Tagline:    d.Tagline,
		Rating:     d.VoteAverage,
		Runtime:    d.Runtime,
		Cast:       castNames(d.Credits.Cast),
		CastIDs:    castIDs(d.Credits.Cast),
		Genres:     genreNames(d.Genres),
		Director:   name,
		DirectorID: dirID,
		TrailerKey: d.Videos.trailerKey(),
		IMDbID:     d.ImdbID,
	}
}
