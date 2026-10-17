package bukanir

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockBackend serves TMDB, TPB, and EZTV endpoints from one server.
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

	// EZTV
	mux.HandleFunc("/api/get-torrents", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"torrents":[{"title":"Breaking.Bad.S05E15.720p.HDTV.x264","magnet_url":"magnet:?xt=urn:btih:ZZZZ","hash":"ZZZZ","seeds":80,"size_bytes":"550000000","season":"5","episode":"15"}]}`))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c, err := New(context.Background(), Config{CacheDir: t.TempDir(), TMDBKey: "k", CacheTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	c.tmdbBase, c.tpbBase, c.eztvBase = srv.URL, srv.URL, srv.URL
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

func TestSearchEpisodesMergesEZTV(t *testing.T) {
	c := mockBackend(t)
	movies, err := c.Search(context.Background(), "breaking bad", SearchOptions{Media: "episodes", SortBy: "episodes"})
	if err != nil {
		t.Fatal(err)
	}
	// TPB contributes S05E14, EZTV contributes S05E15.
	if len(movies) != 2 {
		t.Fatalf("got %d episodes, want 2 (TPB + EZTV)", len(movies))
	}
	if movies[0].Episode != 14 || movies[1].Episode != 15 {
		t.Errorf("episode sort wrong: %d then %d", movies[0].Episode, movies[1].Episode)
	}
	for _, m := range movies {
		if m.Title != "Breaking Bad" || m.Category != CategoryTV {
			t.Errorf("tv result = %+v", m)
		}
	}
}

func TestSearchCaches(t *testing.T) {
	c := mockBackend(t)
	if _, err := c.Search(context.Background(), "matrix", SearchOptions{Media: "movies"}); err != nil {
		t.Fatal(err)
	}
	// Point sources at a dead server; a cache hit must still return results.
	c.tpbBase = "http://127.0.0.1:0"
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
	sortMovies(movies, "seeders")
	if movies[0].Seeders != 50 || movies[2].Seeders != 5 {
		t.Errorf("not sorted by seeders: %+v", movies)
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
