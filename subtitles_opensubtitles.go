package bukanir

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const openSubtitlesBaseURL = "https://api.opensubtitles.com/api/v1"

// openSubtitlesUserAgent identifies the app to OpenSubtitles.
var openSubtitlesUserAgent = "bukanir v" + Version

type openSubtitlesProvider struct{ c *Client }

func (openSubtitlesProvider) name() string { return "opensubtitles" }

type osSearchResponse struct {
	Data []osSubtitleData `json:"data"`
}

type osSubtitleData struct {
	Attributes osSubtitleAttributes `json:"attributes"`
}

type osSubtitleAttributes struct {
	Language        string           `json:"language"`
	DownloadCount   int              `json:"download_count"`
	Ratings         float64          `json:"ratings"`
	Release         string           `json:"release"`
	HearingImpaired bool             `json:"hearing_impaired"`
	Files           []osFile         `json:"files"`
	FeatureDetails  osFeatureDetails `json:"feature_details"`
}

type osFile struct {
	FileID   int    `json:"file_id"`
	FileName string `json:"file_name"`
}

type osFeatureDetails struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
}

type osLoginResponse struct {
	Token string `json:"token"`
}

type osDownloadResponse struct {
	Link     string `json:"link"`
	FileName string `json:"file_name"`
}

func (p openSubtitlesProvider) search(ctx context.Context, q SubtitleQuery) ([]Subtitle, error) {
	id := strings.TrimPrefix(q.IMDbID, "tt")
	if id == "" {
		return nil, nil
	}
	vals := url.Values{"imdb_id": {id}}
	if q.Language != "" {
		vals.Set("languages", q.Language)
	}
	if q.isEpisode() {
		vals.Set("season_number", strconv.Itoa(q.Season))
		vals.Set("episode_number", strconv.Itoa(q.Episode))
	}

	var resp osSearchResponse
	if err := p.c.doJSON(ctx, http.MethodGet, p.c.subsBase+"/subtitles?"+vals.Encode(), p.headers(""), nil, &resp); err != nil {
		return nil, err
	}

	subs := make([]Subtitle, 0, len(resp.Data))
	for _, d := range resp.Data {
		a := d.Attributes
		if len(a.Files) == 0 {
			continue
		}
		subs = append(subs, Subtitle{
			Provider:        "opensubtitles",
			FileID:          a.Files[0].FileID,
			Language:        a.Language,
			Release:         a.Release,
			Title:           a.FeatureDetails.Title,
			Year:            a.FeatureDetails.Year,
			Rating:          a.Ratings,
			Downloads:       a.DownloadCount,
			HearingImpaired: a.HearingImpaired,
		})
	}
	return subs, nil
}

func (p openSubtitlesProvider) download(ctx context.Context, sub Subtitle) ([]byte, string, error) {
	token, err := p.login(ctx)
	if err != nil {
		return nil, "", err
	}

	var dl osDownloadResponse
	req := map[string]int{"file_id": sub.FileID}
	if err := p.c.doJSON(ctx, http.MethodPost, p.c.subsBase+"/download", p.headers(token), req, &dl); err != nil {
		return nil, "", err
	}
	if dl.Link == "" {
		return nil, "", errors.New("bukanir: OpenSubtitles returned no download link")
	}

	data, err := p.c.getBytes(ctx, dl.Link)
	if err != nil {
		return nil, "", err
	}
	return gunzipIfNeeded(data), dl.FileName, nil
}

// login returns a cached OpenSubtitles token, logging in on first use.
func (p openSubtitlesProvider) login(ctx context.Context) (string, error) {
	c := p.c
	c.subsMu.Lock()
	defer c.subsMu.Unlock()
	if c.subsToken != "" {
		return c.subsToken, nil
	}
	if c.subsErr != nil {
		return "", c.subsErr
	}
	token, err := p.doLogin(ctx)
	if err != nil {
		c.subsErr = err
		return "", err
	}
	c.subsToken = token
	return token, nil
}

func (p openSubtitlesProvider) doLogin(ctx context.Context) (string, error) {
	c := p.c
	user, pass := c.cfg.openSubtitlesUser(), c.cfg.openSubtitlesPass()
	if user == "" || pass == "" {
		return "", errors.New("bukanir: OpenSubtitles credentials required to download subtitles")
	}
	var resp osLoginResponse
	body := map[string]string{"username": user, "password": pass}
	if err := c.doJSON(ctx, http.MethodPost, c.subsBase+"/login", p.headers(""), body, &resp); err != nil {
		return "", err
	}
	if resp.Token == "" {
		return "", errors.New("bukanir: OpenSubtitles login returned no token")
	}
	return resp.Token, nil
}

func (p openSubtitlesProvider) headers(token string) map[string]string {
	h := map[string]string{
		"Api-Key":    p.c.cfg.openSubtitlesKey(),
		"User-Agent": openSubtitlesUserAgent,
		"Accept":     "application/json",
	}
	if token != "" {
		h["Authorization"] = "Bearer " + token
	}
	return h
}

func gunzipIfNeeded(data []byte) []byte {
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		return data
	}
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return data
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		return data
	}
	return out
}
