package bukanir

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const osSearchBody = `{"data":[
	{"attributes":{"language":"en","download_count":10,"ratings":6.0,"release":"Breaking.Bad.S05E14.WEB.x264","files":[{"file_id":111,"file_name":"a.srt"}],"feature_details":{"title":"Breaking Bad","year":2013}}},
	{"attributes":{"language":"en","download_count":99,"ratings":8.0,"release":"Breaking.Bad.S05E14.720p.HDTV.x264","files":[{"file_id":222,"file_name":"b.srt"}],"feature_details":{"title":"Breaking Bad","year":2013}}},
	{"attributes":{"language":"en","release":"no files here","files":[]}}
]}`

func mockOpenSubtitles(t *testing.T, loginHits, apiHits *int32) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/subtitles", func(w http.ResponseWriter, r *http.Request) {
		if apiHits != nil {
			atomic.AddInt32(apiHits, 1)
		}
		if r.Header.Get("Api-Key") == "" {
			t.Error("search missing Api-Key")
		}
		if r.URL.Query().Get("imdb_id") != "0903747" {
			t.Errorf("imdb_id = %q", r.URL.Query().Get("imdb_id"))
		}
		w.Write([]byte(osSearchBody))
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if loginHits != nil {
			atomic.AddInt32(loginHits, 1)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["username"] == "" || body["password"] == "" {
			t.Error("login missing credentials")
		}
		w.Write([]byte(`{"token":"tok123"}`))
	})

	c, err := New(context.Background(), Config{
		CacheDir:          t.TempDir(),
		OpenSubtitlesKey:  "apikey",
		OpenSubtitlesUser: "user",
		OpenSubtitlesPass: "pass",
	})
	if err != nil {
		t.Fatal(err)
	}

	var srv *httptest.Server
	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		if apiHits != nil {
			atomic.AddInt32(apiHits, 1)
		}
		if r.Header.Get("Authorization") != "Bearer tok123" {
			t.Errorf("download missing bearer: %q", r.Header.Get("Authorization"))
		}
		w.Write([]byte(`{"link":"` + srv.URL + `/file","file_name":"sub.srt"}`))
	})
	mux.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		zw.Write([]byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n"))
		zw.Close()
		w.Write(buf.Bytes())
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c.subsBase = srv.URL
	t.Cleanup(func() { c.Close() })
	return c
}

func TestSubtitlesSearchScoredSorted(t *testing.T) {
	c := mockOpenSubtitles(t, nil, nil)
	subs, err := c.Subtitles(context.Background(), SubtitleQuery{
		IMDbID: "tt0903747", Language: "en", Release: "Breaking.Bad.S05E14.720p.HDTV.x264",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 2 {
		t.Fatalf("got %d subtitles, want 2 (empty-files dropped)", len(subs))
	}
	if subs[0].FileID != 222 {
		t.Errorf("best match FileID = %d, want 222", subs[0].FileID)
	}
	if subs[0].Score <= subs[1].Score {
		t.Errorf("scores not sorted: %.3f <= %.3f", subs[0].Score, subs[1].Score)
	}
	if subs[0].Title != "Breaking Bad" || subs[0].Year != 2013 {
		t.Errorf("metadata = %q/%d", subs[0].Title, subs[0].Year)
	}
}

func TestDownloadSubtitle(t *testing.T) {
	var logins int32
	c := mockOpenSubtitles(t, &logins, nil)
	data, name, err := c.DownloadSubtitle(context.Background(), Subtitle{Provider: "opensubtitles", FileID: 222})
	if err != nil {
		t.Fatal(err)
	}
	if name != "sub.srt" {
		t.Errorf("filename = %q", name)
	}
	if !bytes.Contains(data, []byte("Hello")) {
		t.Errorf("subtitle content not decompressed: %q", data)
	}

	if _, _, err := c.DownloadSubtitle(context.Background(), Subtitle{Provider: "opensubtitles", FileID: 111}); err != nil {
		t.Fatal(err)
	}
	if logins != 1 {
		t.Errorf("logged in %d times, want 1", logins)
	}
}

func TestDownloadSubtitleNoCreds(t *testing.T) {
	c, err := New(context.Background(), Config{CacheDir: t.TempDir(), OpenSubtitlesKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, _, err := c.DownloadSubtitle(context.Background(), Subtitle{Provider: "opensubtitles", FileID: 1}); err == nil {
		t.Fatal("expected error without credentials")
	}
}

func TestSubtitlesNoIMDb(t *testing.T) {
	c := testClient(t)
	// No imdb id and no episode: nothing to search, yields no results.
	subs, err := c.Subtitles(context.Background(), SubtitleQuery{Language: "en", Release: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("got %d subtitles, want 0", len(subs))
	}
}

func TestJaro(t *testing.T) {
	cases := []struct {
		a, b string
		want float64
	}{
		{"", "", 1},
		{"abc", "abc", 1},
		{"abc", "xyz", 0},
		{"MARTHA", "MARHTA", 0.944444},
		{"DIXON", "DICKSONX", 0.766666},
	}
	for _, tc := range cases {
		got := jaro(tc.a, tc.b)
		if math.Abs(got-tc.want) > 0.0001 {
			t.Errorf("jaro(%q,%q) = %.6f, want %.6f", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestJaroEmpty(t *testing.T) {
	if jaro("abc", "") != 0 || jaro("", "abc") != 0 {
		t.Error("empty vs non-empty should be 0")
	}
}

func TestSubtitlesCached(t *testing.T) {
	var hits int32
	c := mockOpenSubtitles(t, nil, &hits)
	c.cache = newCache(t.TempDir(), time.Hour)
	q := SubtitleQuery{IMDbID: "tt0903747", Language: "en", Release: "Breaking.Bad.S05E14.720p.HDTV.x264"}
	for range 2 {
		subs, err := c.Subtitles(context.Background(), q)
		if err != nil {
			t.Fatal(err)
		}
		if len(subs) != 2 || subs[0].FileID != 222 {
			t.Fatalf("got %d subtitles, best %d", len(subs), subs[0].FileID)
		}
	}
	if hits != 1 {
		t.Errorf("provider searched %d times, want 1", hits)
	}
}

func TestDownloadSubtitleCached(t *testing.T) {
	var hits int32
	c := mockOpenSubtitles(t, nil, &hits)
	c.subsCache = newCache(t.TempDir(), time.Hour)
	for range 2 {
		data, name, err := c.DownloadSubtitle(context.Background(), Subtitle{Provider: "opensubtitles", FileID: 222})
		if err != nil {
			t.Fatal(err)
		}
		if name != "sub.srt" || !bytes.Contains(data, []byte("Hello")) {
			t.Fatalf("got %q %q", name, data)
		}
	}
	if hits != 1 {
		t.Errorf("provider downloaded %d times, want 1", hits)
	}
}

func TestSubdlLive(t *testing.T) {
	if testing.Short() {
		t.Skip("contacts SubDL; skipped under -short")
	}
	if os.Getenv("BUKANIR_SUBDL_KEY") == "" {
		t.Skip("BUKANIR_SUBDL_KEY not set")
	}
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	subs, err := subdlProvider{c}.search(ctx, SubtitleQuery{IMDbID: "tt0063350", Title: "Night of the Living Dead", Year: 1968, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) == 0 {
		t.Fatal("no SubDL subtitles")
	}
	best := subs[0]
	if best.Format == "" || !strings.Contains(best.DownloadURI, "/subtitle/") {
		t.Fatalf("unexpected unpacked file: %+v", best)
	}
	data, name, err := c.DownloadSubtitle(ctx, best)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 1000 || !strings.HasSuffix(name, "."+best.Format) || bytes.HasPrefix(data, zipMagic) {
		t.Fatalf("got %d bytes as %q", len(data), name)
	}
}

func TestMovieSubtitlesSearch500(t *testing.T) {
	c := testClient(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/search.php", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<a href="/movie-1.html">Blade Runner 2049 (2017)</a><a href="/movie-2.html">Runner (2018)</a><a href="/movie-3.html">Front Runner, The (2018)</a>`))
	})
	mux.HandleFunc("/movie-3.html", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<table><tr><td><a href="/subtitle-42.html"><div class="subtitle"><div><img src="images/flags/en.gif" alt="en"><a href="/subtitle-42.html"><b>Front Runner, The 2018 english subtitles (1080p.WEB)</b></a></div></div></a></td></tr>` +
			`<tr><td><a href="/subtitle-43.html"><div class="subtitle"><div><img src="images/flags/fr.gif" alt="fr"><a href="/subtitle-43.html"><b>Front Runner, The 2018 french subtitles ()</b></a></div></div></a></td></tr></table>`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c.movieSubtitlesBase = srv.URL
	p := movieSubtitlesProvider{c}

	subs, err := p.search(context.Background(), SubtitleQuery{Title: "The Front Runner", Year: 2018, Language: "en"})
	if err != nil || len(subs) != 1 || subs[0].DownloadURI != "/download-42.html" || subs[0].Release != "1080p.WEB" {
		t.Fatalf("search = %+v, %v", subs, err)
	}
	for _, q := range []SubtitleQuery{{Title: "Runner", Year: 2026}, {Title: "Blade Runner", Year: 2017}} {
		if path, err := p.findMovie(context.Background(), q.Title, q.Year); err != nil || path != "" {
			t.Errorf("findMovie(%q, %d) = %q, %v; want no match", q.Title, q.Year, path, err)
		}
	}
	if path, _ := p.findMovie(context.Background(), "Runner", 2018); path != "/movie-2.html" {
		t.Errorf("findMovie(Runner, 2018) = %q", path)
	}
}
