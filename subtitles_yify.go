package bukanir

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const yifyBaseURL = "https://yifysubtitles.ch"

type yifyProvider struct{ c *Client }

func (yifyProvider) name() string { return "yify" }

func (p yifyProvider) search(ctx context.Context, q SubtitleQuery) ([]Subtitle, error) {
	if q.isEpisode() || q.IMDbID == "" {
		return nil, nil
	}
	want := languageName(q.Language)

	doc, err := p.c.fetchDocHeaders(ctx, p.c.yifyBase+"/movie-imdb/"+q.IMDbID, browserHeaders(""))
	if err != nil {
		return nil, err
	}

	var subs []Subtitle
	doc.Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
		lang := strings.TrimSpace(row.Find(".sub-lang").Text())
		if want != "" && !strings.EqualFold(lang, want) {
			return
		}
		href, ok := row.Find("a[href^='/subtitles/']").Attr("href")
		if !ok {
			return
		}
		rating, _ := strconv.ParseFloat(strings.TrimSpace(row.Find(".rating-cell").Text()), 64)
		slug := strings.TrimPrefix(href, "/subtitles/")
		subs = append(subs, Subtitle{
			Provider:    "yify",
			DownloadURI: "/subtitle/" + slug + ".zip",
			Language:    q.Language,
			Release:     slug,
			Title:       q.Title,
			Year:        q.Year,
			Rating:      rating,
		})
	})
	return subs, nil
}

func (p yifyProvider) download(ctx context.Context, sub Subtitle) ([]byte, string, error) {
	uri := sub.DownloadURI
	if !strings.HasPrefix(uri, "http") {
		uri = p.c.yifyBase + uri
	}
	slug := strings.TrimSuffix(strings.TrimPrefix(sub.DownloadURI, "/subtitle/"), ".zip")
	referer := p.c.yifyBase + "/subtitles/" + slug
	data, err := p.c.doBytes(ctx, http.MethodGet, uri, browserHeaders(referer), nil)
	if err != nil {
		return nil, "", err
	}
	out, ext := unzipSubtitle(data)
	return out, sanitizeFilename(sub.Title) + ext, nil
}
