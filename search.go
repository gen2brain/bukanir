package bukanir

import (
	"cmp"
	"context"
	"encoding/json"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

// torrentioConcurrency bounds the parallel per-episode Torrentio requests.
const torrentioConcurrency = 4

var enrichConcurrency = 3 * runtime.NumCPU()

// SearchOptions tunes a Search.
type SearchOptions struct {
	// Media selects sources: "all" (default), "movies", or "episodes".
	Media string
	// SortBy orders results: "seeders" (default) or "episodes".
	SortBy string
	// Limit caps the number of results. Zero means no limit.
	Limit int
}

func (o SearchOptions) withDefaults() SearchOptions {
	if o.Media == "" {
		o.Media = "all"
	}
	if o.SortBy == "" {
		o.SortBy = "seeders"
	}
	return o
}

// Search finds movies and TV episodes for a query.
func (c *Client) Search(ctx context.Context, query string, opts SearchOptions) ([]TMovie, error) {
	opts = opts.withDefaults()

	cacheKey := c.cache.key("search", query, opts.Media)
	if data, ok := c.cache.get(cacheKey); ok {
		var cached []TMovie
		if json.Unmarshal(data, &cached) == nil {
			return finalizeMovies(cached, opts), nil
		}
	}

	torrents, err := c.gatherTorrents(ctx, query, opts.Media)
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
	preferRealSeeders(movies)
	movies = dedupeMovies(movies)

	if data, err := json.Marshal(movies); err == nil {
		_ = c.cache.put(cacheKey, data)
	}
	return finalizeMovies(movies, opts), nil
}

// gatherTorrents collects candidate torrents from the sources for media.
func (c *Client) gatherTorrents(ctx context.Context, query, media string) ([]torrentResult, error) {
	var (
		mu       sync.Mutex
		all      []torrentResult
		firstErr error
		wg       sync.WaitGroup
	)
	collect := func(res []torrentResult, err error) {
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return
		}
		all = append(all, res...)
	}

	wg.Go(func() {
		collect(c.tpbSearch(ctx, query, media))
	})

	if media == "all" || media == "movies" {
		wg.Go(func() {
			res, _ := c.ytsSearch(ctx, query)
			collect(res, nil)
		})
	}

	if media == "all" || media == "episodes" {
		wg.Go(func() {
			// Episode source failures are non-fatal.
			show, ok := c.showForQuery(ctx, query)
			if !ok {
				return
			}
			var sources sync.WaitGroup
			sources.Go(func() {
				if show.season > 0 && show.episode > 0 {
					res, _ := c.eztvEpisode(ctx, show.imdbID, show.season, show.episode)
					collect(res, nil)
					return
				}
				res, _ := c.eztvSearch(ctx, show.imdbID, eztvPageSize)
				collect(res, nil)
			})
			sources.Go(func() {
				collect(c.torrentioShow(ctx, show, media == "episodes"), nil)
			})
			sources.Wait()
		})
	}
	wg.Wait()

	if len(all) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return all, nil
}

// showMatch is the TV show a query names, plus the episode when the query has SxxEyy.
type showMatch struct {
	id      int
	name    string
	imdbID  string
	season  int
	episode int
}

type episodeRef struct{ season, episode int }

// showForQuery resolves a query to a TV show and its IMDb id via TMDB.
func (c *Client) showForQuery(ctx context.Context, query string) (showMatch, bool) {
	season, episode := releaseSeasonEpisode(query)
	title := strings.TrimSpace(reSeason.ReplaceAllString(query, ""))
	if title == "" {
		title = query
	}
	resp, err := c.tmdbSearchTV(ctx, title)
	if err != nil || len(resp.Results) == 0 {
		return showMatch{}, false
	}
	r := resp.Results[0]
	ext, err := c.tmdbTVExternalIDs(ctx, r.ID)
	if err != nil || ext.ImdbID == "" {
		return showMatch{}, false
	}
	return showMatch{id: r.ID, name: r.title(), imdbID: ext.ImdbID, season: season, episode: episode}, true
}

// torrentioShow asks Torrentio for the query's episode, or for the latest season's aired episodes when latest is set.
func (c *Client) torrentioShow(ctx context.Context, show showMatch, latest bool) []torrentResult {
	var eps []episodeRef
	switch {
	case show.season > 0 && show.episode > 0:
		eps = []episodeRef{{show.season, show.episode}}
	case latest:
		eps = c.latestSeasonEpisodes(ctx, show.id)
	}
	lists := boundedMap(ctx, eps, torrentioConcurrency, func(ctx context.Context, e episodeRef) ([]torrentResult, bool) {
		res, err := c.torrentioSearch(ctx, show.imdbID, show.name, e.season, e.episode)
		return res, err == nil && len(res) > 0
	})
	var out []torrentResult
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

// latestSeasonEpisodes lists the aired episodes of a show's most recent season.
func (c *Client) latestSeasonEpisodes(ctx context.Context, id int) []episodeRef {
	details, err := c.tmdbTVDetails(ctx, id, 0)
	if err != nil || details.LastEpisode == nil || details.LastEpisode.SeasonNumber <= 0 {
		return nil
	}
	season, err := c.tmdbTVDetails(ctx, id, details.LastEpisode.SeasonNumber)
	if err != nil {
		return nil
	}
	today := time.Now().Format("2006-01-02")
	var eps []episodeRef
	for _, e := range season.Episodes {
		if e.EpisodeNumber > 0 && e.AirDate != "" && e.AirDate <= today {
			eps = append(eps, episodeRef{details.LastEpisode.SeasonNumber, e.EpisodeNumber})
		}
	}
	return eps
}

func finalizeMovies(movies []TMovie, opts SearchOptions) []TMovie {
	SortMovies(movies, opts.SortBy)
	if opts.Limit > 0 && len(movies) > opts.Limit {
		movies = movies[:opts.Limit]
	}
	return movies
}

// preferRealSeeders replaces the seeder count of YTS entries, which the YTS API
// caps, with the best count another source reports for the same title.
func preferRealSeeders(movies []TMovie) {
	type key struct{ id, season, episode int }
	real := map[key]int{}
	for _, m := range movies {
		if m.Source == "yts" {
			continue
		}
		k := key{m.ID, m.Season, m.Episode}
		if m.Seeders > real[k] {
			real[k] = m.Seeders
		}
	}
	for i := range movies {
		if movies[i].Source != "yts" {
			continue
		}
		if n, ok := real[key{movies[i].ID, movies[i].Season, movies[i].Episode}]; ok {
			movies[i].Seeders = n
		}
	}
}

// dedupeMovies keeps one entry per title, episode and quality: the best seeded,
// and on a tie the one whose seeder count is not a YTS estimate.
func dedupeMovies(movies []TMovie) []TMovie {
	type key struct {
		id, season, episode int
		quality             string
	}
	index := map[key]int{}
	out := make([]TMovie, 0, len(movies))
	for _, m := range movies {
		k := key{m.ID, m.Season, m.Episode, m.Quality}
		i, seen := index[k]
		if !seen {
			index[k] = len(out)
			out = append(out, m)
			continue
		}
		cur := out[i]
		if m.Seeders > cur.Seeders || (m.Seeders == cur.Seeders && cur.Source == "yts" && m.Source != "yts") {
			out[i] = m
		}
	}
	return out
}

// SortMovies orders movies by "seeders" (default) or "episodes". Episodes keeps each
// title together, titles in seeders order, newest episode first within a title.
func SortMovies(movies []TMovie, sortBy string) {
	if sortBy != "episodes" {
		slices.SortStableFunc(movies, func(a, b TMovie) int { return cmp.Compare(b.Seeders, a.Seeders) })
		return
	}
	type title struct {
		tv   bool
		id   int
		name string
	}
	keyOf := func(m TMovie) title {
		if m.ID == 0 {
			return title{tv: m.Category.IsTV(), name: m.Title}
		}
		return title{tv: m.Category.IsTV(), id: m.ID}
	}
	best := map[title]int{}
	first := map[title]int{}
	for i, m := range movies {
		k := keyOf(m)
		if _, seen := first[k]; !seen {
			first[k] = i
		}
		best[k] = max(best[k], m.Seeders)
	}
	slices.SortStableFunc(movies, func(a, b TMovie) int {
		ka, kb := keyOf(a), keyOf(b)
		if ka != kb {
			if c := cmp.Compare(best[kb], best[ka]); c != 0 {
				return c
			}
			return cmp.Compare(first[ka], first[kb])
		}
		if c := cmp.Compare(b.Season, a.Season); c != 0 {
			return c
		}
		if c := cmp.Compare(b.Episode, a.Episode); c != 0 {
			return c
		}
		return cmp.Compare(b.Seeders, a.Seeders)
	})
}

// boundedMap applies fn with at most limit concurrent calls, keeping results where fn returns true.
func boundedMap[T, R any](ctx context.Context, items []T, limit int, fn func(context.Context, T) (R, bool)) []R {
	if limit < 1 {
		limit = 1
	}
	var (
		sem = make(chan struct{}, limit)
		mu  sync.Mutex
		out []R
		wg  sync.WaitGroup
	)
	for _, it := range items {
		select {
		case <-ctx.Done():
			wg.Wait()
			return out
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(it T) {
			defer wg.Done()
			defer func() { <-sem }()
			if r, ok := fn(ctx, it); ok {
				mu.Lock()
				out = append(out, r)
				mu.Unlock()
			}
		}(it)
	}
	wg.Wait()
	return out
}

func itemsFromResults(results []tmdbResult) []TItem {
	items := make([]TItem, 0, len(results))
	for _, r := range results {
		if r.title() == "" {
			continue
		}
		items = append(items, TItem{Title: r.title(), Year: r.year()})
	}
	return items
}
