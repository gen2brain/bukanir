package bukanir

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// maxTorrentSize is the upper bound on a candidate torrent.
const maxTorrentSize = 5 << 30 // 5 GiB

// defaultTrackers are appended to magnet links.
var defaultTrackers = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.demonii.com:1337/announce",
	"udp://open.stealth.si:80/announce",
	"udp://open.tracker.cl:1337/announce",
	"udp://exodus.desync.com:6969/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://tracker.qu.ax:6969/announce",
	"udp://tracker.bittor.pw:1337/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
	"https://tracker.tamersunion.org:443/announce",
}

// torrentResult is a candidate torrent from a source, before TMDB enrichment.
type torrentResult struct {
	Name      string // raw torrent name
	Title     string // cleaned title for matching
	Magnet    string
	Year      int
	Size      int64
	SizeHuman string
	Seeders   int
	Category  Category
	Season    int
	Episode   int
	Quality   string
	IMDbID    string // set by sources that know it, skips the title match
	Source    string // "tpb", "eztv", "torrentio" or "yts"
	FileName  string // episode file inside a multi-file torrent
	FileIdx   int    // its index in the torrent
	Pack      bool   // Size may be the episode file or the whole pack
}

func magnetLink(infoHash, name string) string {
	var b strings.Builder
	b.WriteString("magnet:?xt=urn:btih:")
	b.WriteString(infoHash)
	b.WriteString("&dn=")
	b.WriteString(url.QueryEscape(name))
	for _, tr := range defaultTrackers {
		b.WriteString("&tr=")
		b.WriteString(url.QueryEscape(tr))
	}
	return b.String()
}

func humanizeBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// DefaultTPBHost is the apibay API host used when Config.TPBHost is empty.
const DefaultTPBHost = "apibay.org"

// tpbBaseURL builds the API root for host.
func tpbBaseURL(host string) string {
	if host == "" {
		host = DefaultTPBHost
	}
	if strings.HasSuffix(host, ".onion") {
		return "http://" + host
	}
	return "https://" + host
}

var tpbMediaCategories = map[string]string{
	"all":      "201,207,205,208",
	"movies":   "201,207",
	"episodes": "205,208",
}

// tpbTorrent is one entry of an apibay JSON response.
type tpbTorrent struct {
	ID       json.Number `json:"id"`
	InfoHash string      `json:"info_hash"`
	Category json.Number `json:"category"`
	Name     string      `json:"name"`
	Size     json.Number `json:"size"`
	Seeders  json.Number `json:"seeders"`
}

func (c *Client) tpbSearch(ctx context.Context, query, media string) ([]torrentResult, error) {
	cats, ok := tpbMediaCategories[media]
	if !ok {
		cats = tpbMediaCategories["all"]
	}
	u := fmt.Sprintf("%s/q.php?q=%s&cat=%s", c.tpbBase, url.QueryEscape(query), cats)
	body, err := c.getBytes(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseTPB(body)
}

func (c *Client) tpbTop(ctx context.Context, category Category) ([]torrentResult, error) {
	u := fmt.Sprintf("%s/precompiled/data_top100_%d.json", c.tpbBase, category)
	body, err := c.getBytes(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseTPB(body)
}

func parseTPB(body []byte) ([]torrentResult, error) {
	var raw []tpbTorrent
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	results := make([]torrentResult, 0, len(raw))
	for _, t := range raw {
		id, _ := t.ID.Int64()
		if id == 0 {
			continue // apibay's "no results" sentinel
		}
		seeders, _ := t.Seeders.Int64()
		if seeders == 0 {
			continue
		}
		size, _ := t.Size.Int64()
		if size <= 0 || size > maxTorrentSize {
			continue
		}
		cat64, _ := t.Category.Int64()
		category := Category(cat64)

		rel := parseRelease(t.Name)
		if rel.Title == "" {
			continue
		}
		if category.IsTV() && rel.Season == 0 && rel.Episode == 0 {
			continue
		}

		results = append(results, torrentResult{
			Source:    "tpb",
			Name:      t.Name,
			Title:     rel.Title,
			Magnet:    magnetLink(t.InfoHash, t.Name),
			Year:      rel.Year,
			Size:      size,
			SizeHuman: humanizeBytes(size),
			Seeders:   int(seeders),
			Category:  category,
			Season:    rel.Season,
			Episode:   rel.Episode,
			Quality:   rel.Quality,
		})
	}
	return results, nil
}

// DefaultYTSHost is the YTS API host used when Config.YTSHost is empty.
const DefaultYTSHost = "yts.gg"

// ytsBaseURL builds the API root for host.
func ytsBaseURL(host string) string {
	if host == "" {
		host = DefaultYTSHost
	}
	if strings.HasSuffix(host, ".onion") {
		return "http://" + host
	}
	return "https://" + host
}

type ytsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Movies []ytsMovie `json:"movies"`
	} `json:"data"`
}

type ytsMovie struct {
	Title     string       `json:"title_english"`
	TitleLong string       `json:"title_long"`
	Year      int          `json:"year"`
	IMDbID    string       `json:"imdb_code"`
	Torrents  []ytsTorrent `json:"torrents"`
}

type ytsTorrent struct {
	Hash      string `json:"hash"`
	Quality   string `json:"quality"`
	Type      string `json:"type"`
	Seeds     int    `json:"seeds"`
	SizeBytes int64  `json:"size_bytes"`
}

func (c *Client) ytsSearch(ctx context.Context, query string) ([]torrentResult, error) {
	u := fmt.Sprintf("%s/api/v2/list_movies.json?query_term=%s&limit=50&sort_by=seeds", c.ytsBase, url.QueryEscape(query))
	body, err := c.getBytes(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseYTS(body)
}

func (c *Client) ytsTop(ctx context.Context, category Category) ([]torrentResult, error) {
	u := c.ytsBase + "/api/v2/list_movies.json?limit=50&sort_by=seeds"
	if category == CategoryHDMovies {
		u += "&quality=1080p"
	}
	body, err := c.getBytes(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseYTS(body)
}

func parseYTS(body []byte) ([]torrentResult, error) {
	var resp ytsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "ok" {
		return nil, fmt.Errorf("bukanir: yts status %q", resp.Status)
	}
	var results []torrentResult
	for _, m := range resp.Data.Movies {
		if m.Title == "" || m.IMDbID == "" {
			continue
		}
		for _, t := range m.Torrents {
			if t.Seeds == 0 || t.Hash == "" || t.SizeBytes <= 0 || t.SizeBytes > maxTorrentSize {
				continue
			}
			name := fmt.Sprintf("%s [%s] [%s] [YTS]", m.TitleLong, t.Quality, strings.ToUpper(t.Type))
			quality := strings.TrimSuffix(t.Quality, "p")
			category := CategoryHDMovies
			if quality == "720" || quality == "480" {
				category = CategoryMovies
			}
			results = append(results, torrentResult{
				Source:    "yts",
				Name:      name,
				Title:     strings.ToLower(m.Title),
				Magnet:    magnetLink(t.Hash, name),
				Year:      m.Year,
				Size:      t.SizeBytes,
				SizeHuman: humanizeBytes(t.SizeBytes),
				Seeders:   t.Seeds,
				Category:  category,
				Quality:   quality,
				IMDbID:    m.IMDbID,
			})
		}
	}
	return results, nil
}

// eztvBaseURL is the EZTV API root.
const eztvBaseURL = "https://eztvx.to"

// eztvTorrent is one entry of an EZTV get-torrents response.
type eztvTorrent struct {
	Title     string      `json:"title"`
	MagnetURL string      `json:"magnet_url"`
	Hash      string      `json:"hash"`
	Seeds     int         `json:"seeds"`
	SizeBytes json.Number `json:"size_bytes"`
	Season    json.Number `json:"season"`
	Episode   json.Number `json:"episode"`
}

// eztvPageSize is the largest page the EZTV API returns.
const eztvPageSize = 100

// eztvMaxPages caps how many pages an episode lookup reads.
const eztvMaxPages = 10

// eztvSearch returns a show's newest TV torrents for its IMDb id.
func (c *Client) eztvSearch(ctx context.Context, imdbID string, limit int) ([]torrentResult, error) {
	res, _, err := c.eztvPage(ctx, imdbID, limit, 1)
	return res, err
}

// eztvEpisode reads a show's EZTV pages, which are ordered by date added, and keeps one episode.
func (c *Client) eztvEpisode(ctx context.Context, imdbID string, season, episode int) ([]torrentResult, error) {
	first, count, err := c.eztvPage(ctx, imdbID, eztvPageSize, 1)
	if err != nil {
		return nil, err
	}
	pages := min((count+eztvPageSize-1)/eztvPageSize, eztvMaxPages)
	var rest []int
	for p := 2; p <= pages; p++ {
		rest = append(rest, p)
	}
	lists := boundedMap(ctx, rest, 4, func(ctx context.Context, page int) ([]torrentResult, bool) {
		res, _, err := c.eztvPage(ctx, imdbID, eztvPageSize, page)
		return res, err == nil
	})
	var out []torrentResult
	for _, list := range append([][]torrentResult{first}, lists...) {
		for _, tr := range list {
			if tr.Season == season && tr.Episode == episode {
				out = append(out, tr)
			}
		}
	}
	return out, nil
}

// eztvPage fetches one page of a show's EZTV torrents and the show's total torrent count.
func (c *Client) eztvPage(ctx context.Context, imdbID string, limit, page int) ([]torrentResult, int, error) {
	id := strings.TrimPrefix(imdbID, "tt")
	if id == "" {
		return nil, 0, nil
	}
	if limit <= 0 || limit > eztvPageSize {
		limit = eztvPageSize
	}
	u := fmt.Sprintf("%s/api/get-torrents?imdb_id=%s&limit=%d&page=%d", c.eztvBase, id, limit, page)
	body, err := c.getBytes(ctx, u)
	if err != nil {
		return nil, 0, err
	}
	var head struct {
		Count int `json:"torrents_count"`
	}
	_ = json.Unmarshal(body, &head)
	res, err := parseEZTV(body)
	return res, head.Count, err
}

// parseEZTV decodes an EZTV response and applies the candidate filters.
func parseEZTV(body []byte) ([]torrentResult, error) {
	var resp struct {
		Torrents json.RawMessage `json:"torrents"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(resp.Torrents))
	if trimmed == "" || trimmed == "false" || trimmed == "null" {
		return nil, nil
	}

	var torrents []eztvTorrent
	if err := json.Unmarshal(resp.Torrents, &torrents); err != nil {
		return nil, err
	}

	results := make([]torrentResult, 0, len(torrents))
	for _, t := range torrents {
		if t.Seeds == 0 {
			continue
		}
		size, _ := t.SizeBytes.Int64()
		if size <= 0 || size > maxTorrentSize {
			continue
		}
		rel := parseRelease(t.Title)
		if rel.Title == "" {
			continue
		}
		// Prefer EZTV's explicit season/episode over the parsed name.
		season, episode := rel.Season, rel.Episode
		if v, err := t.Season.Int64(); err == nil && v > 0 {
			season = int(v)
		}
		if v, err := t.Episode.Int64(); err == nil && v > 0 {
			episode = int(v)
		}
		if season == 0 && episode == 0 {
			continue
		}
		magnet := t.MagnetURL
		if magnet == "" && t.Hash != "" {
			magnet = magnetLink(t.Hash, t.Title)
		}
		if magnet == "" {
			continue
		}

		results = append(results, torrentResult{
			Source:    "eztv",
			Name:      t.Title,
			Title:     rel.Title,
			Magnet:    magnet,
			Year:      rel.Year,
			Size:      size,
			SizeHuman: humanizeBytes(size),
			Seeders:   t.Seeds,
			Category:  CategoryTV,
			Season:    season,
			Episode:   episode,
			Quality:   rel.Quality,
		})
	}
	return results, nil
}

// torrentioBaseURL is the Torrentio add-on root.
const torrentioBaseURL = "https://torrentio.strem.fun"

// torrentioStream is one entry of a Torrentio stream response.
type torrentioStream struct {
	Title         string `json:"title"`
	InfoHash      string `json:"infoHash"`
	FileIdx       *int   `json:"fileIdx"`
	BehaviorHints struct {
		Filename string `json:"filename"`
	} `json:"behaviorHints"`
}

var (
	reTorrentioSeeders = regexp.MustCompile(`👤 (\d+)`)
	reTorrentioSize    = regexp.MustCompile(`💾 ([\d.]+) ([KMGT]?B)`)
)

// torrentioSearch returns the torrents Torrentio lists for one episode of a show.
func (c *Client) torrentioSearch(ctx context.Context, imdbID, show string, season, episode int) ([]torrentResult, error) {
	if imdbID == "" || season <= 0 || episode <= 0 {
		return nil, nil
	}
	if !strings.HasPrefix(imdbID, "tt") {
		imdbID = "tt" + imdbID
	}
	u := fmt.Sprintf("%s/stream/series/%s:%d:%d.json", c.torrentioBase, imdbID, season, episode)
	body, err := c.getBytes(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseTorrentio(body, show, season, episode)
}

// parseTorrentio decodes a Torrentio response for one episode; a torrent not named after it is a season pack.
func parseTorrentio(body []byte, show string, season, episode int) ([]torrentResult, error) {
	var resp struct {
		Streams []torrentioStream `json:"streams"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	results := make([]torrentResult, 0, len(resp.Streams))
	for _, st := range resp.Streams {
		if st.InfoHash == "" {
			continue
		}
		lines := strings.Split(st.Title, "\n")
		name := strings.TrimSpace(lines[0])
		meta := strings.Join(lines[1:], " ")

		seeders := 0
		if m := reTorrentioSeeders.FindStringSubmatch(meta); m != nil {
			seeders, _ = strconv.Atoi(m[1])
		}
		if seeders == 0 {
			continue
		}
		size := torrentioSize(meta)
		if size <= 0 {
			continue
		}

		rel := parseRelease(name)
		pack := rel.Season != season || rel.Episode != episode
		file := ""
		if len(lines) > 2 && !strings.HasPrefix(strings.TrimSpace(lines[1]), "👤") {
			file = st.BehaviorHints.Filename
		}
		if pack && file == "" {
			continue
		}
		if !pack && size > maxTorrentSize {
			continue
		}

		quality := rel.Quality
		if file != "" {
			if q := releaseQuality(file); q != "" {
				quality = q
			}
		}
		human := humanizeBytes(size)
		if pack {
			human = "season pack"
		}
		idx := 0
		if file != "" && st.FileIdx != nil {
			idx = *st.FileIdx
		}
		title := show
		if title == "" {
			title = rel.Title
		}

		results = append(results, torrentResult{
			Source:    "torrentio",
			Name:      name,
			Title:     title,
			Magnet:    magnetLink(st.InfoHash, name),
			Year:      rel.Year,
			Size:      size,
			SizeHuman: human,
			Seeders:   seeders,
			Category:  CategoryTV,
			Season:    season,
			Episode:   episode,
			Quality:   quality,
			FileName:  file,
			FileIdx:   idx,
			Pack:      pack,
		})
	}
	return results, nil
}

// torrentioSize reads the size Torrentio prints after the 💾 marker.
func torrentioSize(meta string) int64 {
	m := reTorrentioSize.FindStringSubmatch(meta)
	if m == nil {
		return 0
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	mult := map[string]float64{"B": 1, "KB": 1 << 10, "MB": 1 << 20, "GB": 1 << 30, "TB": 1 << 40}[m[2]]
	return int64(v * mult)
}

// Poster size indices into TMDB's poster_sizes list.
const (
	posterSmall  = 0
	posterMedium = 1
	posterLarge  = 3
	posterXLarge = 4
)

// enrich turns a torrent into a TMovie by matching it against TMDB.
func (c *Client) enrich(ctx context.Context, conf *tmdbConfiguration, tr torrentResult) (TMovie, bool) {
	if tr.Category.IsTV() {
		return c.enrichTV(ctx, conf, tr)
	}
	return c.enrichMovie(ctx, conf, tr)
}

func (c *Client) enrichMovie(ctx context.Context, conf *tmdbConfiguration, tr torrentResult) (TMovie, bool) {
	if tr.IMDbID != "" {
		if found, err := c.tmdbFind(ctx, tr.IMDbID); err == nil && len(found.MovieResults) > 0 {
			r := found.MovieResults[0]
			return buildMovie(conf, r, r.PosterPath, tr), true
		}
	}
	resp, err := c.tmdbSearchMovie(ctx, tr.Title)
	if err != nil {
		return TMovie{}, false
	}
	match, ok := pickByYear(resp.Results, tr.Year)
	if !ok {
		return TMovie{}, false
	}
	return buildMovie(conf, match, match.PosterPath, tr), true
}

func (c *Client) enrichTV(ctx context.Context, conf *tmdbConfiguration, tr torrentResult) (TMovie, bool) {
	resp, err := c.tmdbSearchTV(ctx, tr.Title)
	if err != nil || len(resp.Results) == 0 {
		return TMovie{}, false
	}
	match := resp.Results[0]
	poster := match.PosterPath
	if tr.Season > 0 {
		if imgs, err := c.tmdbTVImages(ctx, match.ID, tr.Season); err == nil && len(imgs.Posters) > 0 {
			poster = imgs.Posters[0].FilePath
		}
	}
	return buildMovie(conf, match, poster, tr), true
}

// pickByYear returns the first TMDB result within one year of the torrent's year.
func pickByYear(results []tmdbResult, year int) (tmdbResult, bool) {
	if len(results) == 0 {
		return tmdbResult{}, false
	}
	if year == 0 {
		return results[0], true
	}
	for _, r := range results {
		if ry := r.year(); ry != 0 && abs(ry-year) <= 1 {
			return r, true
		}
	}
	return tmdbResult{}, false
}

func buildMovie(conf *tmdbConfiguration, r tmdbResult, posterPath string, tr torrentResult) TMovie {
	year := r.year()
	if year == 0 {
		year = tr.Year
	}
	return TMovie{
		ID:           r.ID,
		Title:        r.title(),
		Year:         year,
		PosterSmall:  conf.posterURL(posterPath, posterSmall),
		PosterMedium: conf.posterURL(posterPath, posterMedium),
		PosterLarge:  conf.posterURL(posterPath, posterLarge),
		PosterXLarge: conf.posterURL(posterPath, posterXLarge),
		Size:         tr.Size,
		SizeHuman:    tr.SizeHuman,
		Seeders:      tr.Seeders,
		Magnet:       tr.Magnet,
		Release:      tr.Name,
		Source:       tr.Source,
		Category:     tr.Category,
		Season:       tr.Season,
		Episode:      tr.Episode,
		Quality:      tr.Quality,
		FileName:     tr.FileName,
		FileIdx:      tr.FileIdx,
		Pack:         tr.Pack,
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Release-name heuristics extracting title, year, season/episode, and quality.
var (
	reYear     = regexp.MustCompile(`19\d{2}|20\d{2}`)
	reQuality  = regexp.MustCompile(`(720|1080)p?`)
	reTitleTag = regexp.MustCompile(
		`(?i)(.*?)(dvdrip|xvid|dvdscr|brrip|bdrip|divx|klaxxon|hc|webrip|hdrip|camrip|hdtv|eztv|proper|x264|x265|hevc|480p|720p|1080p|2160p|[\*\{\(\[]?19\d{2}|20\d{2}).*`)
	reTitleParen = regexp.MustCompile(`(.*?)\(.*\)(.*)`)
	reSeason     = regexp.MustCompile(`(?i)(?:s|season)(\d{2})(?:e|x|episode)(\d{2}).*`)
)

type release struct {
	Title   string // cleaned, lower-cased title for TMDB matching
	Year    int    // 0 if unknown
	Season  int    // 0 if not an episode
	Episode int    // 0 if not an episode
	Quality string // "720", "1080", or ""
}

func parseRelease(name string) release {
	season, episode := releaseSeasonEpisode(name)
	return release{
		Title:   cleanTitle(name),
		Year:    releaseYear(name),
		Season:  season,
		Episode: episode,
		Quality: releaseQuality(name),
	}
}

// cleanTitle reduces a torrent name to a bare title.
func cleanTitle(name string) string {
	title := strings.ToLower(strings.TrimSuffix(name, "\n"))
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, ":", "")

	// Multi-episode packs repeat the season marker; drop them before extracting the title.
	if m := reSeason.FindAllStringSubmatch(title, -1); len(m) > 1 {
		title = reSeason.ReplaceAllString(title, " ")
	}
	if m := reTitleTag.FindStringSubmatch(title); m != nil {
		title = m[1]
	}
	if m := reTitleParen.FindStringSubmatch(title); m != nil {
		title = m[1]
	}
	title = strings.ReplaceAll(title, "(", "")
	title = strings.ReplaceAll(title, ")", "")
	title = reSeason.ReplaceAllString(title, "")
	return strings.TrimSpace(title)
}

func releaseYear(name string) int {
	if m := reYear.FindString(name); m != "" {
		y, _ := strconv.Atoi(m)
		return y
	}
	return 0
}

func releaseQuality(name string) string {
	if m := reQuality.FindStringSubmatch(name); m != nil {
		return m[1]
	}
	return ""
}

func releaseSeasonEpisode(name string) (season, episode int) {
	if m := reSeason.FindStringSubmatch(name); m != nil {
		season, _ = strconv.Atoi(m[1])
		episode, _ = strconv.Atoi(m[2])
	}
	return season, episode
}
