package bukanir

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

const gestdownBaseURL = "https://api.gestdown.info"

type gestdownProvider struct{ c *Client }

func (gestdownProvider) name() string { return "gestdown" }

type gdShowsResponse struct {
	Shows []gdShow `json:"shows"`
}

type gdShow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	TVDBID int    `json:"tvDbId"`
	TMDBID int    `json:"tmdbId"`
}

type gdSubtitlesResponse struct {
	MatchingSubtitles []gdSubtitle `json:"matchingSubtitles"`
}

type gdSubtitle struct {
	SubtitleID      string `json:"subtitleId"`
	Version         string `json:"version"`
	DownloadURI     string `json:"downloadUri"`
	Language        string `json:"language"`
	DownloadCount   int    `json:"downloadCount"`
	HD              bool   `json:"hd"`
	HearingImpaired bool   `json:"hearingImpaired"`
	Completed       bool   `json:"completed"`
}

func (p gestdownProvider) search(ctx context.Context, q SubtitleQuery) ([]Subtitle, error) {
	if !q.isEpisode() || q.Title == "" {
		return nil, nil
	}
	lang := languageName(q.Language)
	if lang == "" {
		return nil, nil
	}
	showID, err := p.findShow(ctx, q)
	if err != nil || showID == "" {
		return nil, err
	}

	u := fmt.Sprintf("%s/subtitles/get/%s/%d/%d/%s",
		p.c.gestdownBase, showID, q.Season, q.Episode, url.PathEscape(lang))
	var resp gdSubtitlesResponse
	if err := p.c.getJSON(ctx, u, &resp); err != nil {
		return nil, err
	}

	subs := make([]Subtitle, 0, len(resp.MatchingSubtitles))
	for _, m := range resp.MatchingSubtitles {
		if !m.Completed {
			continue
		}
		subs = append(subs, Subtitle{
			Provider:        "gestdown",
			DownloadURI:     m.DownloadURI,
			Language:        q.Language,
			Release:         m.Version,
			Title:           q.Title,
			Year:            q.Year,
			Downloads:       m.DownloadCount,
			HD:              m.HD,
			HearingImpaired: m.HearingImpaired,
		})
	}
	return subs, nil
}

// findShow resolves the show to a gestdown id.
func (p gestdownProvider) findShow(ctx context.Context, q SubtitleQuery) (string, error) {
	u := fmt.Sprintf("%s/shows/search/%s", p.c.gestdownBase, url.PathEscape(q.Title))
	var resp gdShowsResponse
	if err := p.c.getJSON(ctx, u, &resp); err != nil {
		return "", err
	}
	if q.TMDBID != 0 {
		for _, s := range resp.Shows {
			if s.TMDBID == q.TMDBID {
				return s.ID, nil
			}
		}
	}
	if len(resp.Shows) > 0 {
		return resp.Shows[0].ID, nil
	}
	return "", nil
}

func (p gestdownProvider) download(ctx context.Context, sub Subtitle) ([]byte, string, error) {
	uri := sub.DownloadURI
	if !strings.HasPrefix(uri, "http") {
		uri = p.c.gestdownBase + uri
	}
	data, err := p.c.getBytes(ctx, uri)
	if err != nil {
		return nil, "", err
	}
	name := sanitizeFilename(sub.Title) + ".srt"
	return data, name, nil
}
