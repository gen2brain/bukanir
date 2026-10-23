package bukanir

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const tpbMatrix = `{"id":"123","info_hash":"AAAA","name":"The.Matrix.1999.1080p.x264","size":"1500000000","seeders":"100","category":"207"}`

// A response exercising every filter.
const tpbBody = `[` +
	tpbMatrix + `,` +
	`{"id":"0","name":"No results returned","info_hash":"0","size":"0","seeders":"0","category":"0"},` +
	`{"id":"5","info_hash":"BBBB","name":"Some.Movie.2020","size":"1000","seeders":"0","category":"201"},` +
	`{"id":"6","info_hash":"CCCC","name":"Big.Movie.2020","size":"9999999999999","seeders":"5","category":"201"},` +
	`{"id":"7","info_hash":"DDDD","name":"Show.Without.Episode.2020","size":"1000000","seeders":"5","category":"205"},` +
	`{"id":"8","info_hash":"EEEE","name":"Breaking.Bad.S05E14.HDTV.x264","size":"500000000","seeders":"50","category":"205"}` +
	`]`

const eztvBody = `{"torrents_count":3,"torrents":[
	{"title":"Breaking.Bad.S05E14.720p.HDTV.x264","magnet_url":"magnet:?xt=urn:btih:EEEE","hash":"EEEE","seeds":50,"size_bytes":"500000000","season":"5","episode":"14"},
	{"title":"Some.Show.S01E01","magnet_url":"","hash":"FFFF","seeds":0,"size_bytes":"1000000","season":"1","episode":"1"},
	{"title":"Big.Show.S01E01","magnet_url":"magnet:?xt=urn:btih:GGGG","hash":"GGGG","seeds":10,"size_bytes":"9999999999999","season":"1","episode":"1"}
]}`

const torrentioBody = `{"streams":[
	{"title":"Breaking.Bad.S01E05.1080p.WEB.x264\n👤 30 💾 1.5 GB ⚙️ ThePirateBay","infoHash":"aaaa","behaviorHints":{"filename":"Breaking.Bad.S01E05.1080p.WEB.x264.mkv"}},
	{"title":"Breaking.Bad.S01.720p.BluRay\nBreaking.Bad.S01E05.720p.BluRay.mkv\n👤 80 💾 12.5 GB ⚙️ 1337x","infoHash":"bbbb","fileIdx":4,"behaviorHints":{"filename":"Breaking.Bad.S01E05.720p.BluRay.mkv"}},
	{"title":"Breaking.Bad.S01E05.2160p.WEB\n👤 20 💾 9.1 GB ⚙️ ThePirateBay","infoHash":"cccc","behaviorHints":{}},
	{"title":"Breaking.Bad.S01E05.480p\n👤 0 💾 300 MB ⚙️ ThePirateBay","infoHash":"dddd","behaviorHints":{}},
	{"title":"Breaking.Bad.Complete\n👤 5 💾 40 GB ⚙️ RARBG","infoHash":"eeee","behaviorHints":{}},
	{"title":"Breaking.Bad.S01E05-E07.480p\nBreaking.Bad.S01E05.480p.mkv\n👤 7 💾 900 MB ⚙️ 1337x","infoHash":"ffff","fileIdx":0,"behaviorHints":{"filename":"Breaking.Bad.S01E05.480p.mkv"}}
]}`

const ytsBody = `{"status":"ok","data":{"movies":[
	{"title_english":"The Matrix","title_long":"The Matrix (1999)","year":1999,"imdb_code":"tt0133093","torrents":[
		{"hash":"AAAA","quality":"720p","type":"bluray","seeds":40,"size_bytes":800000000},
		{"hash":"BBBB","quality":"1080p","type":"web","seeds":90,"size_bytes":1600000000},
		{"hash":"CCCC","quality":"2160p","type":"bluray","seeds":5,"size_bytes":9999999999999},
		{"hash":"DDDD","quality":"1080p","type":"bluray","seeds":0,"size_bytes":1600000000}]},
	{"title_english":"","title_long":"Nameless (2000)","year":2000,"imdb_code":"tt1","torrents":[{"hash":"EEEE","quality":"720p","type":"web","seeds":9,"size_bytes":1000}]}
]}}`

func TestMagnetLink(t *testing.T) {
	m := magnetLink("ABCDEF0123", "The Matrix 1999")
	if !strings.HasPrefix(m, "magnet:?xt=urn:btih:ABCDEF0123") {
		t.Errorf("bad prefix: %s", m)
	}
	if !strings.Contains(m, "dn=The+Matrix+1999") {
		t.Errorf("display name not escaped: %s", m)
	}
	if got := strings.Count(m, "&tr="); got != len(defaultTrackers) {
		t.Errorf("trackers = %d, want %d", got, len(defaultTrackers))
	}
}

func TestHumanizeBytes(t *testing.T) {
	cases := map[int64]string{
		512:            "512 B",
		1024:           "1.0 KiB",
		1536:           "1.5 KiB",
		1 << 20:        "1.0 MiB",
		int64(5) << 30: "5.0 GiB",
	}
	for n, want := range cases {
		if got := humanizeBytes(n); got != want {
			t.Errorf("humanizeBytes(%d) = %q, want %q", n, got, want)
		}
	}
}

func mockTPB(t *testing.T, wantCat string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/q.php" && wantCat != "" {
			if got := r.URL.Query().Get("cat"); got != wantCat {
				t.Errorf("cat = %q, want %q", got, wantCat)
			}
		}
		w.Write([]byte(tpbBody))
	}))
	t.Cleanup(srv.Close)

	c, err := New(context.Background(), Config{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	c.tpbBase = srv.URL
	t.Cleanup(func() { c.Close() })
	return c
}

func TestTPBSearchFilters(t *testing.T) {
	c := mockTPB(t, "201,207,205,208")
	res, err := c.tpbSearch(context.Background(), "matrix", "all")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("got %d results, want 2 (kept movie + tv episode)", len(res))
	}

	movie := res[0]
	if movie.Title != "the matrix" || movie.Year != 1999 || movie.Category != CategoryHDMovies || movie.Seeders != 100 {
		t.Errorf("movie = %+v", movie)
	}
	if movie.Magnet == "" || movie.SizeHuman == "" {
		t.Errorf("magnet/size not set: %+v", movie)
	}

	tv := res[1]
	if tv.Category != CategoryTV || tv.Season != 5 || tv.Episode != 14 {
		t.Errorf("tv = %+v", tv)
	}
}

func TestTPBMediaCategoryMapping(t *testing.T) {
	c := mockTPB(t, "201,207")
	if _, err := c.tpbSearch(context.Background(), "x", "movies"); err != nil {
		t.Fatal(err)
	}
}

func TestTPBTop(t *testing.T) {
	c := mockTPB(t, "")
	res, err := c.tpbTop(context.Background(), CategoryMovies)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("top returned %d, want 2", len(res))
	}
}

func TestParseTPBUnescapesNames(t *testing.T) {
	body := []byte(`[{"id":"1","name":"Minions &amp; Monsters (2026) [1080p]","info_hash":"ABC","seeders":"5","size":"1000","category":"207"}]`)
	results, err := parseTPB(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "Minions & Monsters (2026) [1080p]" || strings.Contains(results[0].Title, "amp") {
		t.Fatalf("results = %+v", results)
	}
}

func TestParseTPBBadJSON(t *testing.T) {
	if _, err := parseTPB([]byte("not json")); err == nil {
		t.Fatal("expected error on bad JSON")
	}
}

func mockEZTV(t *testing.T, body string, wantIMDb string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wantIMDb != "" {
			if got := r.URL.Query().Get("imdb_id"); got != wantIMDb {
				t.Errorf("imdb_id = %q, want %q", got, wantIMDb)
			}
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	c, err := New(context.Background(), Config{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	c.eztvBase = srv.URL
	t.Cleanup(func() { c.Close() })
	return c
}

func TestEZTVSearchFilters(t *testing.T) {
	// "tt" prefix must be stripped before querying.
	c := mockEZTV(t, eztvBody, "0903747")
	res, err := c.eztvSearch(context.Background(), "tt0903747", 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("got %d results, want 1 (zero-seeder and oversized dropped)", len(res))
	}
	got := res[0]
	if got.Category != CategoryTV || got.Season != 5 || got.Episode != 14 || got.Seeders != 50 {
		t.Errorf("result = %+v", got)
	}
	if got.Magnet != "magnet:?xt=urn:btih:EEEE" {
		t.Errorf("magnet = %q", got.Magnet)
	}
}

func TestEZTVEpisodePages(t *testing.T) {
	pages := map[string]string{
		"1": `{"torrents_count":150,"torrents":[
			{"title":"Show.S02E01.720p","hash":"A1","seeds":9,"size_bytes":"400000000","season":"2","episode":"1"},
			{"title":"Show.S01E03.720p","hash":"A2","seeds":9,"size_bytes":"400000000","season":"1","episode":"3"}]}`,
		"2": `{"torrents_count":150,"torrents":[
			{"title":"Show.S01E03.1080p","hash":"B1","seeds":4,"size_bytes":"900000000","season":"1","episode":"3"},
			{"title":"Show.S01E02.720p","hash":"B2","seeds":4,"size_bytes":"400000000","season":"1","episode":"2"}]}`,
	}
	var requested sync.Map
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		requested.Store(page, true)
		if r.URL.Query().Get("limit") != "100" {
			t.Errorf("limit = %q", r.URL.Query().Get("limit"))
		}
		w.Write([]byte(pages[page]))
	}))
	t.Cleanup(srv.Close)
	c, err := New(context.Background(), Config{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	c.eztvBase = srv.URL

	res, err := c.eztvEpisode(context.Background(), "tt1234567", 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := requested.Load("2"); !ok {
		t.Error("second page not requested")
	}
	if _, ok := requested.Load("3"); ok {
		t.Error("page past the torrent count requested")
	}
	if len(res) != 2 {
		t.Fatalf("got %d results, want both S01E03 torrents: %+v", len(res), res)
	}
	for _, r := range res {
		if r.Season != 1 || r.Episode != 3 {
			t.Errorf("result = %+v", r)
		}
	}
}

func TestEZTVEmptyTorrentsFalse(t *testing.T) {
	// EZTV returns torrents:false when there are no results.
	c := mockEZTV(t, `{"torrents_count":0,"torrents":false}`, "")
	res, err := c.eztvSearch(context.Background(), "tt1234567", 30)
	if err != nil {
		t.Fatalf("unexpected error on empty response: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("got %d results, want 0", len(res))
	}
}

func TestEZTVEmptyIMDb(t *testing.T) {
	c := mockEZTV(t, eztvBody, "")
	res, err := c.eztvSearch(context.Background(), "", 30)
	if err != nil || res != nil {
		t.Fatalf("empty imdb: got %v, %v", res, err)
	}
}

func TestTorrentioSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stream/series/tt0903747:1:5.json" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(torrentioBody))
	}))
	t.Cleanup(srv.Close)
	c, err := New(context.Background(), Config{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	c.torrentioBase = srv.URL

	res, err := c.torrentioSearch(context.Background(), "0903747", "Breaking Bad", 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	// Oversized single episode, zero seeders and a pack without a file name are dropped.
	if len(res) != 3 {
		t.Fatalf("got %d results, want 3: %+v", len(res), res)
	}
	single, pack, multi := res[0], res[1], res[2]
	if multi.Pack || multi.FileName != "Breaking.Bad.S01E05.480p.mkv" || multi.SizeHuman == "season pack" {
		t.Errorf("multi-episode torrent = %+v", multi)
	}
	if single.Pack || single.Seeders != 30 || single.Size != int64(1.5*(1<<30)) || single.Quality != "1080" || single.FileName != "" || single.Title != "Breaking Bad" {
		t.Errorf("single = %+v", single)
	}
	if single.Season != 1 || single.Episode != 5 || single.Category != CategoryTV || single.Source != "torrentio" {
		t.Errorf("single episode fields = %+v", single)
	}
	if !pack.Pack || pack.FileIdx != 4 || pack.FileName != "Breaking.Bad.S01E05.720p.BluRay.mkv" || pack.Quality != "720" {
		t.Errorf("pack = %+v", pack)
	}
	if pack.Size <= maxTorrentSize || pack.SizeHuman != "season pack" {
		t.Errorf("pack size = %d %q", pack.Size, pack.SizeHuman)
	}
	if !strings.Contains(pack.Magnet, "bbbb") {
		t.Errorf("pack magnet = %q", pack.Magnet)
	}
}

func TestTorrentioSize(t *testing.T) {
	cases := map[string]int64{
		"👤 1 💾 512 MB ⚙️ x": 512 << 20,
		"👤 1 💾 2 GB":        2 << 30,
		"👤 1 💾 700 KB":      700 << 10,
		"no size":           0,
	}
	for in, want := range cases {
		if got := torrentioSize(in); got != want {
			t.Errorf("torrentioSize(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseRelease(t *testing.T) {
	cases := []struct {
		name string
		want release
	}{
		{"The.Matrix.1999.1080p.BluRay.x264-GROUP", release{Title: "the matrix", Year: 1999, Quality: "1080"}},
		{"Game.of.Thrones.S01E02.720p.HDTV.x264", release{Title: "game of thrones", Season: 1, Episode: 2, Quality: "720"}},
		{"Inception (2010) [1080p]", release{Title: "inception", Year: 2010, Quality: "1080"}},
		{"Dune.Part.Two.2024.2160p.WEB-DL.x265", release{Title: "dune part two", Year: 2024, Quality: "2160"}},
		{"Show.S03E04.480p.HDTV", release{Title: "show", Season: 3, Episode: 4, Quality: "480"}},
		{"Show.S03E04.HDTV.x264-KILLERS", release{Title: "show", Season: 3, Episode: 4}},
		{"Breaking.Bad.S05E14.HDTV.x264", release{Title: "breaking bad", Season: 5, Episode: 14}},
		{"Some Random Movie", release{Title: "some random movie"}},
	}
	for _, tc := range cases {
		got := parseRelease(tc.name)
		if got != tc.want {
			t.Errorf("parseRelease(%q) = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestReleaseYearEdge(t *testing.T) {
	if y := releaseYear("No Year Here"); y != 0 {
		t.Errorf("year = %d, want 0", y)
	}
	if y := releaseYear("Movie.2021.x264"); y != 2021 {
		t.Errorf("year = %d, want 2021", y)
	}
}

func mockYTS(t *testing.T, wantQuery string) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/list_movies.json", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("query_term"); got != wantQuery {
			t.Errorf("query_term = %q, want %q", got, wantQuery)
		}
		w.Write([]byte(ytsBody))
	})
	mux.HandleFunc("/configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"images":{"secure_base_url":"https://img/","poster_sizes":["w92","w154","w185","w342","w500"]}}`))
	})
	mux.HandleFunc("/find/tt0133093", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("external_source") != "imdb_id" {
			t.Error("find without external_source")
		}
		w.Write([]byte(`{"movie_results":[{"id":603,"title":"The Matrix","release_date":"1999-03-30","poster_path":"/m.jpg"}]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c, err := New(context.Background(), Config{CacheDir: t.TempDir(), TMDBKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	c.ytsBase, c.tmdbBase = srv.URL, srv.URL
	t.Cleanup(func() { c.Close() })
	return c
}

func TestYTSSearch(t *testing.T) {
	c := mockYTS(t, "matrix")
	res, err := c.ytsSearch(context.Background(), "matrix")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("got %d torrents, want 2 (oversized, unseeded and nameless dropped): %+v", len(res), res)
	}
	sd, hd := res[0], res[1]
	if sd.Category != CategoryMovies || sd.Quality != "720" || hd.Category != CategoryHDMovies || hd.Quality != "1080" {
		t.Errorf("categories/qualities = %v/%s %v/%s", sd.Category, sd.Quality, hd.Category, hd.Quality)
	}
	if hd.Title != "the matrix" || hd.Year != 1999 || hd.IMDbID != "tt0133093" || hd.Seeders != 90 {
		t.Errorf("hd = %+v", hd)
	}
	if hd.Name != "The Matrix (1999) [1080p] [WEB] [YTS]" || !strings.HasPrefix(hd.Magnet, "magnet:?xt=urn:btih:BBBB") {
		t.Errorf("name/magnet = %q %q", hd.Name, hd.Magnet)
	}
}

func TestEnrichMovieByIMDb(t *testing.T) {
	c := mockYTS(t, "matrix")
	res, err := c.ytsSearch(context.Background(), "matrix")
	if err != nil || len(res) == 0 {
		t.Fatal(err)
	}
	conf, err := c.tmdbConfiguration(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	m, ok := c.enrichMovie(context.Background(), conf, res[1])
	if !ok || m.ID != 603 || m.Title != "The Matrix" || m.Magnet != res[1].Magnet || m.PosterLarge == "" {
		t.Fatalf("enriched = %+v (ok=%v)", m, ok)
	}
}
