package bukanir

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	subdlBaseURL      = "https://api.subdl.com/api/v2"
	subdlDownloadBase = "https://dl.subdl.com"
)

type subdlProvider struct{ c *Client }

func (subdlProvider) name() string { return "subdl" }

type sdResponse struct {
	Status    bool         `json:"status"`
	Subtitles []sdSubtitle `json:"subtitles"`
}

type sdSubtitle struct {
	ReleaseName string   `json:"release_name"`
	Lang        string   `json:"lang"`
	URL         string   `json:"url"`
	HI          bool     `json:"hi"`
	Season      int      `json:"season"`
	Episode     *int     `json:"episode"`
	FullSeason  bool     `json:"full_season"`
	Files       []sdFile `json:"unpack_files"`
}

type sdFile struct {
	ReleaseName string `json:"release_name"`
	URL         string `json:"url"`
	Format      string `json:"format"`
	HI          bool   `json:"hi"`
}

func (p subdlProvider) search(ctx context.Context, q SubtitleQuery) ([]Subtitle, error) {
	key := p.c.cfg.subdlKey()
	if key == "" {
		return nil, nil
	}
	vals := url.Values{}
	vals.Set("languages", strings.ToLower(q.Language))
	vals.Set("unpack", "1")
	switch {
	case q.IMDbID != "":
		vals.Set("imdb_id", q.IMDbID)
	case q.TMDBID != 0:
		vals.Set("tmdb_id", strconv.Itoa(q.TMDBID))
	default:
		vals.Set("film_name", q.Title)
	}
	if q.isEpisode() {
		vals.Set("type", "tv")
		vals.Set("season", strconv.Itoa(q.Season))
		vals.Set("episode", strconv.Itoa(q.Episode))
	} else {
		vals.Set("type", "movie")
	}

	headers := browserHeaders("")
	headers["Authorization"] = "Bearer " + key
	var resp sdResponse
	if err := p.c.doJSON(ctx, http.MethodGet, p.c.subdlBase+"/subtitles/search?"+vals.Encode(), headers, nil, &resp); err != nil {
		return nil, err
	}

	subs := make([]Subtitle, 0, len(resp.Subtitles))
	for _, s := range resp.Subtitles {
		if q.isEpisode() && !s.matchesEpisode(q) {
			continue
		}
		sub := Subtitle{
			Provider:        "subdl",
			DownloadURI:     s.URL,
			Language:        q.Language,
			Release:         s.ReleaseName,
			Title:           q.Title,
			Year:            q.Year,
			HearingImpaired: s.HI,
		}
		for _, f := range s.Files {
			if !isSubtitleExt("." + strings.ToLower(f.Format)) {
				continue
			}
			sub.DownloadURI = f.URL
			sub.Format = strings.ToLower(f.Format)
			sub.HearingImpaired = f.HI
			if f.ReleaseName != "" {
				sub.Release = f.ReleaseName
			}
			break
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// matchesEpisode reports whether a subtitle applies to the queried episode.
func (s sdSubtitle) matchesEpisode(q SubtitleQuery) bool {
	if s.Season != 0 && s.Season != q.Season {
		return false
	}
	if s.FullSeason || s.Episode == nil {
		return true
	}
	return *s.Episode == q.Episode
}

var zipMagic = []byte("PK\x03\x04")

func (p subdlProvider) download(ctx context.Context, sub Subtitle) ([]byte, string, error) {
	uri := sub.DownloadURI
	if !strings.HasPrefix(uri, "http") {
		uri = subdlDownloadBase + uri
	}
	if !strings.Contains(uri, "api_key=") {
		sep := "?"
		if strings.Contains(uri, "?") {
			sep = "&"
		}
		uri += sep + "api_key=" + url.QueryEscape(p.c.cfg.subdlKey())
	}
	data, err := p.c.getBytes(ctx, uri)
	if err != nil {
		return nil, "", err
	}
	if sub.Format != "" && !bytes.HasPrefix(data, zipMagic) {
		return data, sanitizeFilename(sub.Title) + "." + sub.Format, nil
	}
	out, ext := unzipSubtitle(data)
	return out, sanitizeFilename(sub.Title) + ext, nil
}
