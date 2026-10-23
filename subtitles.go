package bukanir

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// subtitleProviderTimeout bounds each provider's search within Subtitles.
const subtitleProviderTimeout = 8 * time.Second

// subtitleCacheTTL keeps downloaded subtitle files, which never change upstream.
const subtitleCacheTTL = 180 * 24 * time.Hour

// browserUserAgent is sent to scraping sites that reject non-browser clients.
const browserUserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"

var isoLanguageNames = map[string]string{
	"ar": "Arabic", "bg": "Bulgarian", "cs": "Czech", "da": "Danish",
	"de": "German", "el": "Greek", "en": "English", "es": "Spanish",
	"fa": "Persian", "fi": "Finnish", "fr": "French", "he": "Hebrew",
	"hi": "Hindi", "hr": "Croatian", "hu": "Hungarian", "id": "Indonesian",
	"it": "Italian", "ja": "Japanese", "ko": "Korean", "nl": "Dutch",
	"no": "Norwegian", "pl": "Polish", "pt": "Portuguese", "ro": "Romanian",
	"ru": "Russian", "sk": "Slovak", "sl": "Slovenian", "sr": "Serbian",
	"sv": "Swedish", "th": "Thai", "tr": "Turkish", "uk": "Ukrainian",
	"vi": "Vietnamese", "zh": "Chinese",
}

// SubtitleQuery describes the title to find subtitles for.
type SubtitleQuery struct {
	IMDbID   string
	TMDBID   int // TMDB id
	Title    string
	Year     int
	Season   int
	Episode  int
	Language string // ISO-639-1 code
	Release  string // release name
}

func (q SubtitleQuery) isEpisode() bool { return q.Season > 0 }

// Subtitle is a single subtitle candidate for a title.
type Subtitle struct {
	Provider        string
	FileID          int
	DownloadURI     string
	Language        string
	Release         string
	Title           string
	Year            int
	Rating          float64
	Downloads       int
	HD              bool
	HearingImpaired bool
	Format          string  // file extension without the dot, when the provider reports it
	Score           float64 // release-match similarity in [0,1]
}

type subtitleProvider interface {
	name() string
	search(ctx context.Context, q SubtitleQuery) ([]Subtitle, error)
	download(ctx context.Context, sub Subtitle) ([]byte, string, error)
}

// Subtitles returns the cached or freshly searched candidates for q, best match first.
func (c *Client) Subtitles(ctx context.Context, q SubtitleQuery) ([]Subtitle, error) {
	if q.Language == "" {
		q.Language = "en"
	}

	names := make([]string, 0, len(c.subProviders))
	for _, p := range c.subProviders {
		names = append(names, p.name())
	}
	cacheKey := c.cache.key("subtitles", strings.Join(names, ","), q.IMDbID, strconv.Itoa(q.TMDBID), q.Title,
		strconv.Itoa(q.Year), strconv.Itoa(q.Season), strconv.Itoa(q.Episode), q.Language)
	var all []Subtitle
	if data, ok := c.cache.lookup(ctx, cacheKey); ok {
		_ = json.Unmarshal(data, &all)
	}
	if len(all) == 0 {
		var err error
		if all, err = c.searchSubtitles(ctx, q); err != nil {
			return nil, err
		}
		if data, err := json.Marshal(all); err == nil && len(all) > 0 {
			_ = c.cache.put(cacheKey, data)
		}
	}

	rel := strings.ToLower(q.Release)
	for i := range all {
		all[i].Score = jaro(rel, strings.ToLower(all[i].Release))
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Score != all[j].Score {
			return all[i].Score > all[j].Score
		}
		return all[i].Downloads > all[j].Downloads
	})
	return all, nil
}

func (c *Client) searchSubtitles(ctx context.Context, q SubtitleQuery) ([]Subtitle, error) {
	type result struct {
		subs []Subtitle
		err  error
	}
	ch := make(chan result, len(c.subProviders))
	for _, p := range c.subProviders {
		go func(p subtitleProvider) {
			pctx, cancel := context.WithTimeout(ctx, subtitleProviderTimeout)
			defer cancel()
			subs, err := p.search(pctx, q)
			ch <- result{subs, err}
		}(p)
	}

	var all []Subtitle
	var firstErr error
	for range c.subProviders {
		r := <-ch
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		all = append(all, r.subs...)
	}
	if len(all) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return all, nil
}

type cachedSubtitle struct {
	Name string
	Data []byte
}

// DownloadSubtitle returns a subtitle's content, from the on-disk cache when seen before.
func (c *Client) DownloadSubtitle(ctx context.Context, sub Subtitle) ([]byte, string, error) {
	uri := sub.DownloadURI
	if i := strings.IndexByte(uri, '?'); i >= 0 {
		uri = uri[:i]
	}
	cacheKey := c.subsCache.key("subtitle", sub.Provider, strconv.Itoa(sub.FileID), uri)
	if data, ok := c.subsCache.get(cacheKey); ok {
		var f cachedSubtitle
		if json.Unmarshal(data, &f) == nil && len(f.Data) > 0 {
			return f.Data, f.Name, nil
		}
	}
	for _, p := range c.subProviders {
		if p.name() != sub.Provider {
			continue
		}
		data, name, err := p.download(ctx, sub)
		if err != nil {
			return nil, "", err
		}
		if b, err := json.Marshal(cachedSubtitle{Name: name, Data: data}); err == nil {
			_ = c.subsCache.put(cacheKey, b)
		}
		return data, name, nil
	}
	return nil, "", fmt.Errorf("bukanir: unknown subtitle provider %q", sub.Provider)
}

func browserHeaders(referer string) map[string]string {
	h := map[string]string{"User-Agent": browserUserAgent}
	if referer != "" {
		h["Referer"] = referer
	}
	return h
}

func languageName(code string) string { return isoLanguageNames[strings.ToLower(code)] }

func (c *Client) fetchDocHeaders(ctx context.Context, url string, headers map[string]string) (*goquery.Document, error) {
	body, err := c.doBytes(ctx, http.MethodGet, url, headers, nil)
	if err != nil {
		return nil, err
	}
	return goquery.NewDocumentFromReader(bytes.NewReader(body))
}

// unzipSubtitle returns the first subtitle file inside a zip, or data unchanged if not a zip.
func unzipSubtitle(data []byte) ([]byte, string) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return data, ".srt"
	}
	for _, f := range zr.File {
		ext := strings.ToLower(path.Ext(f.Name))
		if !isSubtitleExt(ext) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		out, err := io.ReadAll(rc)
		rc.Close()
		if err == nil && len(out) > 0 {
			return out, ext
		}
	}
	return data, ".srt"
}

func isSubtitleExt(ext string) bool {
	switch ext {
	case ".srt", ".sub", ".ass", ".ssa", ".vtt":
		return true
	}
	return false
}

func sanitizeFilename(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return r
	}, s)
	if s = strings.TrimSpace(s); s == "" {
		return "subtitle"
	}
	return s
}

// jaro returns the Jaro similarity of a and b in [0, 1].
func jaro(a, b string) float64 {
	if a == b {
		return 1
	}
	la, lb := len(a), len(b)
	if la == 0 || lb == 0 {
		return 0
	}

	matchDist := max(max(la, lb)/2-1, 0)

	aMatched := make([]bool, la)
	bMatched := make([]bool, lb)
	matches := 0
	for i := range la {
		start := max(0, i-matchDist)
		end := min(i+matchDist+1, lb)
		for j := start; j < end; j++ {
			if bMatched[j] || a[i] != b[j] {
				continue
			}
			aMatched[i] = true
			bMatched[j] = true
			matches++
			break
		}
	}
	if matches == 0 {
		return 0
	}

	transpositions := 0
	k := 0
	for i := range la {
		if !aMatched[i] {
			continue
		}
		for !bMatched[k] {
			k++
		}
		if a[i] != b[k] {
			transpositions++
		}
		k++
	}

	m := float64(matches)
	return (m/float64(la) + m/float64(lb) + (m-float64(transpositions)/2)/m) / 3
}
