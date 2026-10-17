package bukanir

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const movieSubtitlesBaseURL = "https://www.moviesubtitles.org"

type movieSubtitlesProvider struct{ c *Client }

func (movieSubtitlesProvider) name() string { return "moviesubtitles" }

func (p movieSubtitlesProvider) search(ctx context.Context, q SubtitleQuery) ([]Subtitle, error) {
	if q.isEpisode() || q.Title == "" {
		return nil, nil
	}

	moviePath, err := p.findMovie(ctx, q.Title, q.Year)
	if err != nil || moviePath == "" {
		return nil, err
	}

	doc, err := p.c.fetchDocHeaders(ctx, p.c.movieSubtitlesBase+moviePath, browserHeaders(""))
	if err != nil {
		return nil, err
	}

	var subs []Subtitle
	doc.Find("div.subtitle").Each(func(_ int, d *goquery.Selection) {
		lang := d.Find("img[alt]").First().AttrOr("alt", "")
		if q.Language != "" && !strings.EqualFold(lang, q.Language) {
			return
		}
		href := d.Find("a[href^='/subtitle-']").First().AttrOr("href", "")
		id := strings.TrimSuffix(strings.TrimPrefix(href, "/subtitle-"), ".html")
		if id == "" {
			return
		}
		subs = append(subs, Subtitle{
			Provider:    "moviesubtitles",
			DownloadURI: "/download-" + id + ".html",
			Language:    q.Language,
			Release:     movieSubtitlesRelease(d.Find("b").First().Text()),
			Title:       q.Title,
			Year:        q.Year,
		})
	})
	return subs, nil
}

// findMovie returns the path of the search result matching title and year, or "" when none does.
// The search page answers 500 Internal Server Error together with a complete result list.
func (p movieSubtitlesProvider) findMovie(ctx context.Context, title string, year int) (string, error) {
	body, err := p.c.doBytesAccept(ctx, http.MethodGet, p.c.movieSubtitlesBase+"/search.php?q="+url.QueryEscape(title),
		browserHeaders(""), nil, func(status int) bool { return status == http.StatusInternalServerError })
	if err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	href := ""
	doc.Find("a[href^='/movie-']").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		if movieSubtitlesMatch(a.Text(), title, year) {
			href = a.AttrOr("href", "")
			return false
		}
		return true
	})
	return href, nil
}

var reMovieSubtitlesResult = regexp.MustCompile(`^(.*?)\s*\((\d{4})\)\s*$`)

// movieSubtitlesMatch reports whether a result like "Front Runner, The (2018)" is title from year, give or take one.
func movieSubtitlesMatch(text, title string, year int) bool {
	name, y := strings.TrimSpace(text), 0
	if m := reMovieSubtitlesResult.FindStringSubmatch(name); m != nil {
		name = m[1]
		y, _ = strconv.Atoi(m[2])
	}
	for _, article := range []string{"The", "A", "An"} {
		if rest, ok := strings.CutSuffix(name, ", "+article); ok {
			name = article + " " + rest
			break
		}
	}
	if !strings.EqualFold(name, strings.TrimSpace(title)) {
		return false
	}
	return year == 0 || y == 0 || (y >= year-1 && y <= year+1)
}

// movieSubtitlesRelease returns the release in "Title 1968 english subtitles (720p.hV)", or the whole text without one.
func movieSubtitlesRelease(text string) string {
	text = strings.TrimSpace(text)
	if i := strings.LastIndex(text, "("); i >= 0 && strings.HasSuffix(text, ")") {
		if r := strings.TrimSpace(text[i+1 : len(text)-1]); r != "" {
			return r
		}
	}
	return text
}

func (p movieSubtitlesProvider) download(ctx context.Context, sub Subtitle) ([]byte, string, error) {
	uri := sub.DownloadURI
	if !strings.HasPrefix(uri, "http") {
		uri = p.c.movieSubtitlesBase + uri
	}
	data, err := p.c.doBytes(ctx, http.MethodGet, uri, browserHeaders(p.c.movieSubtitlesBase), nil)
	if err != nil {
		return nil, "", err
	}
	out, ext := unzipSubtitle(data)
	return out, sanitizeFilename(sub.Title) + ext, nil
}
