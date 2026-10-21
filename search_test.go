package bukanir

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockBackend serves TMDB, TPB, EZTV and Torrentio endpoints from one server.
func mockBackend(t *testing.T) *Client {
	t.Helper()
	mux := http.NewServeMux()

	// TMDB
	mux.HandleFunc("/configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"images":{"secure_base_url":"https://img/","poster_sizes":["w92","w154","w185","w342","w500"]}}`))
	})
	mux.HandleFunc("/search/movie", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"id":603,"title":"The Matrix","release_date":"1999-03-30","poster_path":"/m.jpg"}]}`))
	})
	mux.HandleFunc("/search/tv", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"id":1396,"name":"Breaking Bad","first_air_date":"2008-01-20","poster_path":"/bb.jpg"}]}`))
	})
	mux.HandleFunc("/tv/1396/external_ids", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"imdb_id":"tt0903747"}`))
	})
	mux.HandleFunc("/tv/1396", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":1396,"name":"Breaking Bad","last_episode_to_air":{"season_number":5,"episode_number":16,"air_date":"2013-09-29"}}`))
	})
	mux.HandleFunc("/tv/1396/season/5", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"episodes":[{"season_number":5,"episode_number":15,"air_date":"2013-09-22"},{"season_number":5,"episode_number":16,"air_date":"2013-09-29"},{"season_number":5,"episode_number":17,"air_date":"2999-01-01"}]}`))
	})

	// TPB
	mux.HandleFunc("/q.php", func(w http.ResponseWriter, r *http.Request) {
		cat := r.URL.Query().Get("cat")
		switch cat {
		case "201,207": // movies
			w.Write([]byte(`[` + tpbMatrix + `]`))
		case "205,208": // episodes
			w.Write([]byte(`[{"id":"8","info_hash":"EEEE","name":"Breaking.Bad.S05E14.HDTV.x264","size":"500000000","seeders":"50","category":"205"}]`))
		default: // all
			w.Write([]byte(`[` + tpbMatrix + `]`))
		}
	})

	// YTS
	mux.HandleFunc("/api/v2/list_movies.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","data":{"movies":[]}}`))
	})

	// EZTV
	mux.HandleFunc("/api/get-torrents", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"torrents":[{"title":"Breaking.Bad.S05E15.720p.HDTV.x264","magnet_url":"magnet:?xt=urn:btih:ZZZZ","hash":"ZZZZ","seeds":80,"size_bytes":"550000000","season":"5","episode":"15"}]}`))
	})

	// Torrentio
	mux.HandleFunc("/stream/series/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stream/series/tt0903747:5:16.json":
			w.Write([]byte(`{"streams":[{"title":"Breaking.Bad.S05.1080p.BluRay.x264\nBreaking.Bad.S05E16.1080p.BluRay.x264.mkv\n👤 90 💾 25.4 GB ⚙️ 1337x","infoHash":"pppp","fileIdx":15,"behaviorHints":{"filename":"Breaking.Bad.S05E16.1080p.BluRay.x264.mkv"}}]}`))
		case "/stream/series/tt0903747:5:17.json":
			t.Errorf("unaired episode requested")
		default:
			w.Write([]byte(`{"streams":[]}`))
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c, err := New(context.Background(), Config{CacheDir: t.TempDir(), TMDBKey: "k", CacheTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	c.tmdbBase, c.tpbBase, c.eztvBase, c.ytsBase, c.torrentioBase = srv.URL, srv.URL, srv.URL, srv.URL, srv.URL
	t.Cleanup(func() { c.Close() })
	return c
}

func TestSearchMovies(t *testing.T) {
	c := mockBackend(t)
	movies, err := c.Search(context.Background(), "matrix", SearchOptions{Media: "movies"})
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 {
		t.Fatalf("got %d movies, want 1", len(movies))
	}
	m := movies[0]
	if m.ID != 603 || m.Title != "The Matrix" || m.Year != 1999 {
		t.Errorf("movie = %+v", m)
	}
	if m.PosterMedium != "https://img/w154/m.jpg" {
		t.Errorf("poster = %q", m.PosterMedium)
	}
	if m.Magnet == "" || m.Category != CategoryHDMovies {
		t.Errorf("magnet/category = %q/%d", m.Magnet, m.Category)
	}
}

func TestSearchEpisodesMergesSources(t *testing.T) {
	c := mockBackend(t)
	movies, err := c.Search(context.Background(), "breaking bad", SearchOptions{Media: "episodes", SortBy: "episodes"})
	if err != nil {
		t.Fatal(err)
	}
	// TPB contributes S05E14, EZTV S05E15, Torrentio a season pack for S05E16.
	if len(movies) != 3 {
		t.Fatalf("got %d episodes, want 3 (TPB + EZTV + Torrentio)", len(movies))
	}
	if movies[0].Episode != 16 || movies[1].Episode != 15 || movies[2].Episode != 14 {
		t.Errorf("episode sort wrong: %d, %d, %d", movies[0].Episode, movies[1].Episode, movies[2].Episode)
	}
	for _, m := range movies {
		if m.Title != "Breaking Bad" || m.Category != CategoryTV {
			t.Errorf("tv result = %+v", m)
		}
	}
	pack := movies[0]
	if !pack.Pack || pack.Source != "torrentio" || pack.FileIdx != 15 || pack.FileName != "Breaking.Bad.S05E16.1080p.BluRay.x264.mkv" {
		t.Errorf("pack = %+v", pack)
	}
}

func TestSearchAllAsksTorrentioOnlyForNamedEpisode(t *testing.T) {
	c := mockBackend(t)
	movies, err := c.Search(context.Background(), "breaking bad", SearchOptions{Media: "all"})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range movies {
		if m.Source == "torrentio" {
			t.Errorf("plain query in all media asked Torrentio: %+v", m)
		}
	}

	movies, err = c.Search(context.Background(), "breaking bad s05e16", SearchOptions{Media: "all"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range movies {
		if m.Source == "torrentio" && m.Season == 5 && m.Episode == 16 {
			found = true
		}
	}
	if !found {
		t.Errorf("SxxEyy query did not return the Torrentio episode")
	}
}

func TestShowForQueryStripsEpisode(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/tv":
			query = r.URL.Query().Get("query")
			w.Write([]byte(`{"results":[{"id":1396,"name":"Breaking Bad"}]}`))
		case "/tv/1396/external_ids":
			w.Write([]byte(`{"imdb_id":"tt0903747"}`))
		}
	}))
	t.Cleanup(srv.Close)
	c, err := New(context.Background(), Config{CacheDir: t.TempDir(), TMDBKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	c.tmdbBase = srv.URL

	show, ok := c.showForQuery(context.Background(), "Breaking Bad S02E03 1080p")
	if !ok {
		t.Fatal("no show")
	}
	if query != "breaking bad" {
		t.Errorf("TMDB query = %q, want %q", query, "breaking bad")
	}
	if show.name != "Breaking Bad" || show.imdbID != "tt0903747" || show.season != 2 || show.episode != 3 {
		t.Errorf("show = %+v", show)
	}
}

func TestSearchCaches(t *testing.T) {
	c := mockBackend(t)
	if _, err := c.Search(context.Background(), "matrix", SearchOptions{Media: "movies"}); err != nil {
		t.Fatal(err)
	}
	// Point sources at a dead server; a cache hit must still return results.
	c.tpbBase, c.ytsBase = "http://127.0.0.1:0", "http://127.0.0.1:0"
	movies, err := c.Search(context.Background(), "matrix", SearchOptions{Media: "movies"})
	if err != nil {
		t.Fatalf("cached search failed: %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("cached search got %d", len(movies))
	}
}

func TestBoundedMap(t *testing.T) {
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}
	got := boundedMap(context.Background(), items, 8, func(_ context.Context, n int) (int, bool) {
		return n * 2, n%2 == 0 // keep evens, doubled
	})
	if len(got) != 50 {
		t.Fatalf("got %d results, want 50", len(got))
	}
	sum := 0
	for _, v := range got {
		sum += v
	}
	// 2*(0+2+4+...+98) = 2 * 2450 = 4900
	if sum != 4900 {
		t.Errorf("sum = %d, want 4900", sum)
	}
}

func TestSortMoviesBySeeders(t *testing.T) {
	movies := []TMovie{{Seeders: 5}, {Seeders: 50}, {Seeders: 20}}
	SortMovies(movies, "seeders")
	if movies[0].Seeders != 50 || movies[2].Seeders != 5 {
		t.Errorf("not sorted by seeders: %+v", movies)
	}
}

func TestSortMoviesByEpisodes(t *testing.T) {
	movies := []TMovie{
		{ID: 1, Title: "A", Category: CategoryTV, Season: 1, Episode: 2, Seeders: 10},
		{ID: 2, Title: "B", Category: CategoryTV, Season: 3, Episode: 1, Seeders: 90},
		{ID: 1, Title: "A", Category: CategoryTV, Season: 2, Episode: 1, Seeders: 5},
		{ID: 2, Title: "B", Category: CategoryTV, Season: 3, Episode: 4, Seeders: 20},
		{ID: 1, Title: "A", Category: CategoryHDMovies, Seeders: 40},
		{ID: 1, Title: "A", Category: CategoryTV, Season: 2, Episode: 1, Seeders: 30},
	}
	SortMovies(movies, "episodes")
	type ep struct{ id, season, episode, seeders int }
	want := []ep{{2, 3, 4, 20}, {2, 3, 1, 90}, {1, 0, 0, 40}, {1, 2, 1, 30}, {1, 2, 1, 5}, {1, 1, 2, 10}}
	for i, m := range movies {
		if got := (ep{m.ID, m.Season, m.Episode, m.Seeders}); got != want[i] {
			t.Errorf("movies[%d] = %+v, want %+v", i, got, want[i])
		}
	}
}

func TestListsAndAutocomplete(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/movie/popular", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"id":1,"title":"A","release_date":"2001-01-01"}]}`))
	})
	mux.HandleFunc("/tv/popular", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"id":2,"name":"B","first_air_date":"2002-01-01"}]}`))
	})
	mux.HandleFunc("/genre/movie/list", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"genres":[{"id":28,"name":"Action"}]}`))
	})
	mux.HandleFunc("/search/movie", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"id":3,"title":"Matrix","release_date":"1999-01-01"}]}`))
	})
	mux.HandleFunc("/search/tv", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := New(context.Background(), Config{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.tmdbBase = srv.URL

	pop, err := c.Popular(context.Background())
	if err != nil || len(pop) != 2 || pop[0].Title != "A" || pop[1].Title != "B" {
		t.Fatalf("popular = %+v, %v", pop, err)
	}
	if pop[0].Media != "movie" || pop[1].Media != "tv" {
		t.Fatalf("popular media = %q, %q", pop[0].Media, pop[1].Media)
	}
	genres, err := c.Genres(context.Background())
	if err != nil || len(genres) != 1 || genres[0].Name != "Action" {
		t.Fatalf("genres = %+v, %v", genres, err)
	}
	ac, err := c.Autocomplete(context.Background(), "matr")
	if err != nil || len(ac) != 1 || ac[0].Title != "Matrix" || ac[0].Year != 1999 {
		t.Fatalf("autocomplete = %+v, %v", ac, err)
	}
}

func TestPreferRealSeeders(t *testing.T) {
	movies := []TMovie{
		{ID: 1, Source: "yts", Seeders: 100},
		{ID: 1, Source: "tpb", Seeders: 12},
		{ID: 1, Source: "tpb", Seeders: 40},
		{ID: 2, Source: "yts", Seeders: 100},
		{ID: 3, Source: "yts", Seeders: 100, Season: 1, Episode: 2},
		{ID: 3, Source: "eztv", Seeders: 7, Season: 1, Episode: 3},
	}
	preferRealSeeders(movies)
	want := []int{40, 12, 40, 100, 100, 7}
	for i, m := range movies {
		if m.Seeders != want[i] {
			t.Errorf("movies[%d].Seeders = %d, want %d", i, m.Seeders, want[i])
		}
	}
}

func TestDedupeMovies(t *testing.T) {
	movies := []TMovie{
		{ID: 1, Quality: "1080", Source: "yts", Seeders: 40, Release: "a"},
		{ID: 1, Quality: "1080", Source: "tpb", Seeders: 40, Release: "b"},
		{ID: 1, Quality: "1080", Source: "tpb", Seeders: 90, Release: "c"},
		{ID: 1, Quality: "720", Source: "tpb", Seeders: 5, Release: "d"},
		{ID: 1, Quality: "1080", Season: 1, Episode: 1, Source: "eztv", Seeders: 3, Release: "e"},
		{ID: 2, Quality: "1080", Source: "yts", Seeders: 100, Release: "f"},
	}
	got := dedupeMovies(movies)
	want := []string{"c", "d", "e", "f"}
	if len(got) != len(want) {
		t.Fatalf("got %d movies, want %d: %+v", len(got), len(want), got)
	}
	for i, m := range got {
		if m.Release != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, m.Release, want[i])
		}
	}
	tie := dedupeMovies(movies[:2])
	if len(tie) != 1 || tie[0].Release != "b" {
		t.Errorf("tie = %+v, want the apibay entry", tie)
	}
}
