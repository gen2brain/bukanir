package bukanir

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockTMDB returns a Client pointed at a canned TMDB httptest server.
func mockTMDB(t *testing.T, configHits *int32) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/configuration", func(w http.ResponseWriter, r *http.Request) {
		if configHits != nil {
			atomic.AddInt32(configHits, 1)
		}
		w.Write([]byte(`{"images":{"secure_base_url":"https://img/","poster_sizes":["w92","w154","w185","w342","w500"]}}`))
	})
	mux.HandleFunc("/search/movie", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"page":1,"results":[{"id":603,"title":"The Matrix","release_date":"1999-03-30","poster_path":"/m.jpg","vote_average":8.2}],"total_results":1}`))
	})
	mux.HandleFunc("/search/tv", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"id":1399,"name":"Game of Thrones","first_air_date":"2011-04-17"}]}`))
	})
	mux.HandleFunc("/movie/603", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":603,"imdb_id":"tt0133093","title":"The Matrix","tagline":"Free your mind","runtime":136,"vote_average":8.2,"genres":[{"id":28,"name":"Action"}],"credits":{"cast":[{"id":6384,"name":"Keanu Reeves","character":"Neo"}],"crew":[{"id":905,"name":"Lana Wachowski","job":"Director"}]},"videos":{"results":[{"key":"m8e-FF8MsqU","site":"YouTube","type":"Teaser"},{"key":"vKQi3bBA1y8","site":"YouTube","type":"Trailer"}]}}`))
	})
	mux.HandleFunc("/tv/1399/season/1", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":3624,"episodes":[{"season_number":1,"episode_number":1,"overview":"Ep1","crew":[{"id":1,"name":"Tim Van Patten","job":"Director"}]}]}`))
	})
	mux.HandleFunc("/genre/movie/list", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"genres":[{"id":28,"name":"Action"},{"id":35,"name":"Comedy"}]}`))
	})
	mux.HandleFunc("/discover/movie", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("with_genres") != "28" {
			t.Errorf("discover with_genres = %q, want 28", r.URL.Query().Get("with_genres"))
		}
		w.Write([]byte(`{"results":[{"id":603,"title":"The Matrix","release_date":"1999-03-30"}]}`))
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api_key") == "" {
			t.Errorf("request %s missing api_key", r.URL.Path)
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	c, err := New(context.Background(), Config{CacheDir: t.TempDir(), TMDBKey: "testkey"})
	if err != nil {
		t.Fatal(err)
	}
	c.tmdbBase = srv.URL
	t.Cleanup(func() { c.Close() })
	return c
}

func TestTMDBSearchMovie(t *testing.T) {
	c := mockTMDB(t, nil)
	resp, err := c.tmdbSearchMovie(context.Background(), "matrix")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("got %d results", len(resp.Results))
	}
	r := resp.Results[0]
	if r.title() != "The Matrix" || r.year() != 1999 {
		t.Errorf("title/year = %q/%d", r.title(), r.year())
	}
}

func TestTMDBSearchTV(t *testing.T) {
	c := mockTMDB(t, nil)
	resp, err := c.tmdbSearchTV(context.Background(), "thrones")
	if err != nil {
		t.Fatal(err)
	}
	r := resp.Results[0]
	if r.title() != "Game of Thrones" || r.year() != 2011 {
		t.Errorf("title/year = %q/%d", r.title(), r.year())
	}
}

func TestTMDBMovieDetails(t *testing.T) {
	c := mockTMDB(t, nil)
	d, err := c.tmdbMovieDetails(context.Background(), 603)
	if err != nil {
		t.Fatal(err)
	}
	if d.ImdbID != "tt0133093" || d.Runtime != 136 {
		t.Errorf("details = %+v", d)
	}
	if key := d.Videos.trailerKey(); key != "vKQi3bBA1y8" {
		t.Errorf("trailerKey = %q, want the Trailer, not the Teaser", key)
	}
	name, id := d.Credits.director()
	if name != "Lana Wachowski" || id != 905 {
		t.Errorf("director = %q/%d", name, id)
	}
}

func TestTMDBTVSeasonDetails(t *testing.T) {
	c := mockTMDB(t, nil)
	d, err := c.tmdbTVDetails(context.Background(), 1399, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Episodes) != 1 || d.Episodes[0].Overview != "Ep1" {
		t.Fatalf("episodes = %+v", d.Episodes)
	}
}

func TestTMDBConfigurationMemoized(t *testing.T) {
	var hits int32
	c := mockTMDB(t, &hits)
	for range 3 {
		if _, err := c.tmdbConfiguration(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if hits != 1 {
		t.Fatalf("configuration fetched %d times, want 1", hits)
	}
}

func TestTMDBGenresAndDiscover(t *testing.T) {
	c := mockTMDB(t, nil)
	genres, err := c.tmdbGenres(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(genres) != 2 || genres[0].Name != "Action" {
		t.Fatalf("genres = %+v", genres)
	}
	resp, err := c.tmdbDiscoverMovie(context.Background(), "with_genres", 28, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("discover results = %d", len(resp.Results))
	}
}

func TestPosterURL(t *testing.T) {
	conf := &tmdbConfiguration{Images: tmdbImageConfig{
		SecureBaseURL: "https://img/",
		PosterSizes:   []string{"w92", "w185", "w500"},
	}}
	if got := conf.posterURL("/p.jpg", 1); got != "https://img/w185/p.jpg" {
		t.Errorf("posterURL = %q", got)
	}
	// Out-of-range index clamps to the largest.
	if got := conf.posterURL("/p.jpg", 99); got != "https://img/w500/p.jpg" {
		t.Errorf("posterURL clamp = %q", got)
	}
	// No path -> empty.
	if got := conf.posterURL("", 0); got != "" {
		t.Errorf("posterURL empty path = %q", got)
	}
}

func TestParseYear(t *testing.T) {
	if y := parseYear("", "2011-04-17"); y != 2011 {
		t.Errorf("parseYear fallback = %d", y)
	}
	if y := parseYear("bad", "also-bad"); y != 0 {
		t.Errorf("parseYear invalid = %d", y)
	}
}

func TestTrailerKeyFallback(t *testing.T) {
	// No explicit Trailer -> first YouTube video.
	v := tmdbVideos{Results: []tmdbVideo{
		{Key: "x", Site: "Vimeo", Type: "Trailer"},
		{Key: "y", Site: "YouTube", Type: "Clip"},
	}}
	if got := v.trailerKey(); got != "y" {
		t.Errorf("trailerKey fallback = %q", got)
	}
	if !strings.Contains("YouTube", "You") {
		t.Fatal("sanity")
	}
}

func TestTrailerKeys(t *testing.T) {
	v := tmdbVideos{Results: []tmdbVideo{
		{Key: "clip", Site: "YouTube", Type: "Clip"},
		{Key: "teaser", Site: "YouTube", Type: "Teaser"},
		{Key: "final", Site: "YouTube", Type: "Trailer"},
		{Key: "vimeo", Site: "Vimeo", Type: "Trailer"},
		{Key: "official", Site: "YouTube", Type: "Trailer"},
	}}
	got := v.trailerKeys()
	want := []string{"final", "official", "teaser"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("trailerKeys = %v, want %v", got, want)
	}
	if v.trailerKey() != "final" {
		t.Errorf("trailerKey = %q", v.trailerKey())
	}
}

func mockBrowse(t *testing.T) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"images":{"secure_base_url":"https://img/","poster_sizes":["w92","w154","w185","w342","w500"]}}`))
	})
	mux.HandleFunc("/search/movie", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"id":603,"title":"The Matrix","release_date":"1999-03-30","poster_path":"/m.jpg"}]}`))
	})
	mux.HandleFunc("/discover/movie", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"page":1,"total_pages":1,"results":[{"id":603,"title":"The Matrix","release_date":"1999-03-30","poster_path":"/m.jpg"}]}`))
	})
	// TPB top-100 and search.
	mux.HandleFunc("/precompiled/data_top100_207.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[` + tpbMatrix + `]`))
	})
	mux.HandleFunc("/q.php", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[` + tpbMatrix + `]`))
	})
	mux.HandleFunc("/api/v2/list_movies.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","data":{"movies":[{"title_english":"The Matrix","title_long":"The Matrix (1999)","year":1999,"imdb_code":"tt0133093","torrents":[{"hash":"YTS1","quality":"1080p","type":"bluray","seeds":100,"size_bytes":1600000000}]}]}}`))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c, err := New(context.Background(), Config{CacheDir: t.TempDir(), CacheTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	c.tmdbBase, c.tpbBase, c.ytsBase = srv.URL, srv.URL, srv.URL
	t.Cleanup(func() { c.Close() })
	return c
}

func TestCategory(t *testing.T) {
	c := mockBrowse(t)
	movies, err := c.Category(context.Background(), CategoryHDMovies)
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 || movies[0].Source != "tpb" {
		t.Fatalf("category = %+v, want the two 1080p Matrix entries folded into the apibay one", movies)
	}
	if m := movies[0]; m.Title != "The Matrix" || m.Category != CategoryHDMovies || m.Seeders != 100 {
		t.Errorf("category entry = %+v", m)
	}
}

func TestCategoryInvalid(t *testing.T) {
	c := mockBrowse(t)
	if _, err := c.Category(context.Background(), Category(999)); err == nil {
		t.Fatal("expected error for invalid category")
	}
}

func TestGenreDiscover(t *testing.T) {
	c := mockBrowse(t)
	movies, err := c.Genre(context.Background(), 28, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 {
		t.Fatalf("genre returned %d, want 1", len(movies))
	}
	m := movies[0]
	if m.ID != 603 || m.Title != "The Matrix" || m.Magnet == "" {
		t.Errorf("genre movie = %+v", m)
	}
	if m.PosterMedium != "https://img/w154/m.jpg" {
		t.Errorf("poster = %q", m.PosterMedium)
	}
}

func TestCastAndCrewShareDiscover(t *testing.T) {
	c := mockBrowse(t)
	for _, fn := range []func(context.Context, int, int) ([]TMovie, error){c.Cast, c.Crew} {
		movies, err := fn(context.Background(), 6384, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(movies) != 1 || movies[0].ID != 603 {
			t.Errorf("discover result = %+v", movies)
		}
	}
}

func mockSummary(t *testing.T) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/movie/603", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":603,"imdb_id":"tt0133093","title":"The Matrix","tagline":"Free your mind","runtime":136,"vote_average":8.2,"release_date":"1999-03-30","genres":[{"id":28,"name":"Action"}],"credits":{"cast":[{"id":6384,"name":"Keanu Reeves"}],"crew":[{"id":905,"name":"Lana Wachowski","job":"Director"}]},"videos":{"results":[{"key":"vKQi3bBA1y8","site":"YouTube","type":"Trailer"}]}}`))
	})
	mux.HandleFunc("/tv/1396", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":1396,"name":"Breaking Bad","first_air_date":"2008-01-20","overview":"series overview","genres":[{"id":18,"name":"Drama"}],"credits":{"cast":[{"id":1,"name":"Bryan Cranston"}],"crew":[]},"videos":{"results":[{"key":"k1","site":"YouTube","type":"Trailer"}]}}`))
	})
	mux.HandleFunc("/tv/1396/external_ids", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"imdb_id":"tt0903747"}`))
	})
	mux.HandleFunc("/tv/1396/season/5", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"episodes":[{"season_number":5,"episode_number":14,"overview":"episode overview","crew":[{"id":2,"name":"Ep Director","job":"Director"}]}]}`))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c, err := New(context.Background(), Config{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	c.tmdbBase = srv.URL
	t.Cleanup(func() { c.Close() })
	return c
}

func TestMovieSummary(t *testing.T) {
	c := mockSummary(t)
	s, err := c.Summary(context.Background(), 603, CategoryMovies, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "The Matrix" || s.Year != 1999 || s.Runtime != 136 {
		t.Errorf("summary = %+v", s)
	}
	if s.Director != "Lana Wachowski" || s.DirectorID != 905 {
		t.Errorf("director = %q/%d", s.Director, s.DirectorID)
	}
	if len(s.Cast) != 1 || s.Cast[0] != "Keanu Reeves" || s.CastIDs[0] != 6384 {
		t.Errorf("cast = %+v/%+v", s.Cast, s.CastIDs)
	}
	if s.TrailerKey != "vKQi3bBA1y8" || s.IMDbID != "tt0133093" {
		t.Errorf("trailer/imdb = %q/%q", s.TrailerKey, s.IMDbID)
	}
	if len(s.Genres) != 1 || s.Genres[0] != "Action" {
		t.Errorf("genres = %+v", s.Genres)
	}
}

func TestTVSummaryEpisodeOverlay(t *testing.T) {
	c := mockSummary(t)
	s, err := c.Summary(context.Background(), 1396, CategoryTV, 5, 14)
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "Breaking Bad" || s.IMDbID != "tt0903747" {
		t.Errorf("summary = %+v", s)
	}
	// Episode overview and director must overlay the series values.
	if s.Overview != "episode overview" {
		t.Errorf("overview = %q, want episode overview", s.Overview)
	}
	if s.Director != "Ep Director" || s.DirectorID != 2 {
		t.Errorf("episode director = %q/%d", s.Director, s.DirectorID)
	}
}

func TestDiscoverPrefersRealCounts(t *testing.T) {
	c := mockBrowse(t)
	movies, err := c.Genre(context.Background(), 28, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 || movies[0].Source != "tpb" {
		t.Fatalf("genre pick = %+v, want the apibay torrent", movies)
	}
}

func TestTMDBLookupsMemoized(t *testing.T) {
	var searchHits, imageHits, deniedHits int32
	mux := http.NewServeMux()
	mux.HandleFunc("/configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"images":{"secure_base_url":"https://img/","poster_sizes":["w92","w154","w185","w342","w500"]}}`))
	})
	mux.HandleFunc("/search/tv", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&searchHits, 1)
		time.Sleep(50 * time.Millisecond)
		w.Write([]byte(`{"results":[{"id":1399,"name":"The Terror","first_air_date":"2018-03-25","poster_path":"/s.jpg"}]}`))
	})
	mux.HandleFunc("/tv/1399/season/3/images", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&imageHits, 1)
		http.NotFound(w, r)
	})
	mux.HandleFunc("/tv/1399/season/4/images", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&deniedHits, 1)
		w.WriteHeader(http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c, err := New(context.Background(), Config{CacheDir: t.TempDir(), CacheTTL: time.Hour, TMDBKey: "testkey"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	c.tmdbBase = srv.URL
	ctx := context.Background()
	conf, err := c.tmdbConfiguration(ctx)
	if err != nil {
		t.Fatal(err)
	}

	tr := torrentResult{Title: "The Terror", Season: 3, Episode: 1, Category: CategoryTV}
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if m, ok := c.enrichTV(ctx, conf, tr); !ok || m.ID != 1399 {
				t.Errorf("enrichTV = %+v, %v", m, ok)
			}
		}()
	}
	wg.Wait()
	c.enrichTV(ctx, conf, tr)
	if searchHits != 1 || imageHits != 1 {
		t.Errorf("search hits = %d, image hits = %d, want 1 and 1", searchHits, imageHits)
	}

	tr.Season = 4
	c.enrichTV(ctx, conf, tr)
	c.enrichTV(ctx, conf, tr)
	if deniedHits != 2 {
		t.Errorf("401 hits = %d, want 2 (not cached)", deniedHits)
	}
}

func TestTMDBLookupSurvivesCanceledLeader(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		select {
		case <-time.After(150 * time.Millisecond):
		case <-r.Context().Done():
			return
		}
		w.Write([]byte(`{"id":7}`))
	}))
	t.Cleanup(srv.Close)

	c, err := New(context.Background(), Config{CacheDir: t.TempDir(), CacheTTL: time.Hour, TMDBKey: "testkey"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	c.tmdbBase = srv.URL
	u := c.tmdbURL("/tv/7/external_ids", nil)

	leaderCtx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	leaderErr := make(chan error, 1)
	go func() {
		var out tmdbExternalIDs
		leaderErr <- c.tmdbGetJSON(leaderCtx, u, &out)
	}()
	time.Sleep(10 * time.Millisecond)

	var out struct {
		ID int `json:"id"`
	}
	if err := c.tmdbGetJSON(context.Background(), u, &out); err != nil || out.ID != 7 {
		t.Fatalf("waiter = %+v, %v; want id 7 after the leader's context ended", out, err)
	}
	if err := <-leaderErr; err == nil {
		t.Error("leader with an expired context returned no error")
	}
	if hits != 2 {
		t.Errorf("hits = %d, want 2 (canceled fetch, then the waiter's own)", hits)
	}
}
