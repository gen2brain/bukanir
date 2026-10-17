package bukanir

import (
	"context"
	"encoding/json"
	"runtime"
	"sort"
	"sync"
)

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

var enrichConcurrency = 3 * runtime.NumCPU()

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

	if media == "all" || media == "episodes" {
		wg.Go(func() {
			// EZTV failures are non-fatal.
			res, _ := c.eztvByQuery(ctx, query)
			collect(res, nil)
		})
	}
	wg.Wait()

	if len(all) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return all, nil
}

// eztvByQuery resolves a query to a show's IMDb id via TMDB, then lists EZTV torrents.
func (c *Client) eztvByQuery(ctx context.Context, query string) ([]torrentResult, error) {
	resp, err := c.tmdbSearchTV(ctx, query)
	if err != nil || len(resp.Results) == 0 {
		return nil, err
	}
	ext, err := c.tmdbTVExternalIDs(ctx, resp.Results[0].ID)
	if err != nil || ext.ImdbID == "" {
		return nil, err
	}
	return c.eztvSearch(ctx, ext.ImdbID, 100)
}

func finalizeMovies(movies []TMovie, opts SearchOptions) []TMovie {
	sortMovies(movies, opts.SortBy)
	if opts.Limit > 0 && len(movies) > opts.Limit {
		movies = movies[:opts.Limit]
	}
	return movies
}

func sortMovies(movies []TMovie, sortBy string) {
	if sortBy == "episodes" {
		sort.SliceStable(movies, func(i, j int) bool {
			a, b := movies[i], movies[j]
			if a.Season != b.Season {
				return a.Season < b.Season
			}
			if a.Episode != b.Episode {
				return a.Episode < b.Episode
			}
			return a.Seeders > b.Seeders
		})
		return
	}
	sort.SliceStable(movies, func(i, j int) bool {
		return movies[i].Seeders > movies[j].Seeders
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
