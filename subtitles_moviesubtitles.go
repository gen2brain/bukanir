package bukanir

import (
	"context"
	"net/http"
	"net/url"
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

	moviePath, err := p.findMovie(ctx, q.Title)
	if err != nil || moviePath == "" {
		return nil, err
	}

	doc, err := p.c.fetchDocHeaders(ctx, p.c.movieSubtitlesBase+moviePath, browserHeaders(""))
	if err != nil {
		return nil, err
	}

	var subs []Subtitle
	doc.Find("a.subtitle[href^='/subtitle-']").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		lang := rowLanguage(a)
		if q.Language != "" && !strings.EqualFold(lang, q.Language) {
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(href, "/subtitle-"), ".html")
		if id == "" {
			return
		}
		subs = append(subs, Subtitle{
			Provider:    "moviesubtitles",
			DownloadURI: "/download-" + id + ".html",
			Language:    q.Language,
			Release:     strings.TrimSpace(a.Text()),
			Title:       q.Title,
			Year:        q.Year,
		})
	})
	return subs, nil
}

func (p movieSubtitlesProvider) findMovie(ctx context.Context, title string) (string, error) {
	doc, err := p.c.fetchDocHeaders(ctx, p.c.movieSubtitlesBase+"/search.php?q="+url.QueryEscape(title), browserHeaders(""))
	if err != nil {
		return "", err
	}
	href, _ := doc.Find("a[href^='/movie-']").First().Attr("href")
	return href, nil
}

// rowLanguage finds the 2-letter language code from the flag image in a's row.
func rowLanguage(a *goquery.Selection) string {
	row := a.Closest("tr")
	if row.Length() == 0 {
		row = a.Parent()
	}
	lang := ""
	row.Find("img[alt]").EachWithBreak(func(_ int, img *goquery.Selection) bool {
		if alt := img.AttrOr("alt", ""); len(alt) == 2 {
			lang = alt
			return false
		}
		return true
	})
	return lang
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
