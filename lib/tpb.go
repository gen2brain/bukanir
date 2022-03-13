package bukanir

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/dustin/go-humanize"
)

// tpb type
type tpb struct {
	Host string
}

// tpbTorrent type
type tpbTorrent struct {
	ID       json.Number `json:"id",type:"integer"`
	InfoHash string      `json:"info_hash"`
	Category json.Number `json:"category",type:"integer"`
	Name     string      `json:"name"`
	Size     json.Number `json:"size",type:"integer"`
	Seeders  json.Number `json:"seeders",type:"integer"`
	Leechers json.Number `json:"leechers",type:"integer"`
}

// NewTpb returns new tpb
func NewTpb(host string) *tpb {
	return &tpb{Host: host}
}

// Top returns top torrents for category
func (t *tpb) Top(category int) ([]TTorrent, error) {
	var results []TTorrent
	uri := fmt.Sprintf("http://%s/precompiled/data_top100_%d.json", t.Host, category)

	if verbose {
		log.Printf("TPB: GET %s\n", uri)
	}

	body, err := getBody(uri)
	if err != nil {
		return nil, err
	}

	results, err = t.getTorrents(body)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// Search returns torrents for query
func (t *tpb) Search(query string, page int, cats string) ([]TTorrent, error) {
	var results []TTorrent
	uri := fmt.Sprintf("http://%s/q.php?q=%s&cat=%s", t.Host, url.QueryEscape(query), cats)

	if verbose {
		log.Printf("TPB: GET %s\n", uri)
	}

	body, err := getBody(uri)
	if err != nil {
		return nil, err
	}

	results, err = t.getTorrents(body)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// getTorrents returns torrents from html page
func (t *tpb) getTorrents(body []byte) ([]TTorrent, error) {
	var results []TTorrent
	var torrents []tpbTorrent

	err := json.Unmarshal(body, &torrents)
	if err != nil {
		return nil, err
	}

	for _, torrent := range torrents {
		category, _ := torrent.Category.Int64()
		seeders, _ := torrent.Seeders.Int64()
		size, _ := torrent.Size.Int64()

		if getTitle(torrent.Name) == "" {
			continue
		}

		if seeders == 0 {
			continue
		}

		if size > 5120*1024*1024 {
			continue
		}

		season, _ := strconv.Atoi(getSeason(torrent.Name))
		episode, _ := strconv.Atoi(getEpisode(torrent.Name))
		magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s%s", torrent.InfoHash, url.QueryEscape(torrent.Name), getTrackers())

		if category == CategoryTV || category == CategoryHDTV {
			if season == 0 && episode == 0 {
				continue
			}
		}

		t := TTorrent{
			torrent.Name,
			getTitle(torrent.Name),
			magnet,
			getYear(torrent.Name),
			size,
			humanize.IBytes(uint64(size)),
			int(seeders),
			int(category),
			season,
			episode,
		}

		results = append(results, t)
	}

	return results, nil
}

func getTrackers() string {
	tr := "&tr=" + url.QueryEscape("udp://tracker.coppersurfer.tk:6969/announce")
	tr += "&tr=" + url.QueryEscape("udp://tracker.openbittorrent.com:6969/announce")
	tr += "&tr=" + url.QueryEscape("udp://9.rarbg.me:2780/announce")
	tr += "&tr=" + url.QueryEscape("udp://9.rarbg.to:2710/announce")
	tr += "&tr=" + url.QueryEscape("udp://tracker.opentrackr.org:1337/announce")
	tr += "&tr=" + url.QueryEscape("udp://tracker.torrent.eu.org:451/announce")
	tr += "&tr=" + url.QueryEscape("udp://tracker.tiny-vps.com:6969/announce")
	tr += "&tr=" + url.QueryEscape("udp://open.stealth.si:80/announce")
	tr += "&tr=" + url.QueryEscape("udp://bt1.archive.org:6969/announce")
	tr += "&tr=" + url.QueryEscape("udp://tracker.skynetcloud.site:6969/announce")
	return tr
}
