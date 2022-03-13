package bukanir

import (
	"compress/gzip"
	"context"
	"crypto/md5"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/proxy"
)

var (
	reYear    = regexp.MustCompile(`(.*)(19\d{2}|20\d{2})(.*)`)
	reQuality = regexp.MustCompile(`(.*)(720|1080)p?(.*)`)
	reTitle1  = regexp.MustCompile(`(.*?)(dvdrip|xvid|dvdscr|brrip|bdrip|divx|klaxxon|hc|webrip|hdrip|camrip|hdtv|eztv|proper|x264|480p|720p|1080p|[\*\{\(\[]?[0-9]{4}).*`)
	reTitle2  = regexp.MustCompile(`(.*?)\(.*\)(.*)`)
	reSeason  = regexp.MustCompile(`(?i:s|season)(\d{2})(?i:e|x|episode)(\d{2}).*`)
)

// saveCache saves cache
func saveCache(key string, data []byte, cacheDir string) {
	md5key := md5.Sum([]byte(strings.ToLower(key)))
	file := filepath.Join(cacheDir, fmt.Sprintf("%x.json.gz", md5key))

	err := os.MkdirAll(cacheDir, 0777)
	if err != nil {
		log.Printf("ERROR: MkdirAll %s: %s\n", cacheDir, err.Error())
	}

	f, err := os.Create(file)
	if err != nil {
		log.Printf("ERROR: Create %s: %s\n", file, err.Error())
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	_, err = gz.Write(data)
	if err != nil {
		log.Printf("ERROR: Write: %s\n", err.Error())
	}
}

// getCache gets cache
func getCache(key string, cacheDir string, days int64) []byte {
	md5key := md5.Sum([]byte(strings.ToLower(key)))
	file := filepath.Join(cacheDir, fmt.Sprintf("%x.json.gz", md5key))

	info, err := os.Stat(file)
	if err != nil {
		return nil
	}

	if days != 0 && days > 0 {
		mtime := info.ModTime().Unix()
		if time.Now().Unix()-mtime > 86400*days {
			return nil
		}
	}

	if verbose {
		log.Printf("BUK: Using cache file %s\n", file)
	}

	f, err := os.Open(file)
	if err != nil {
		log.Printf("ERROR: Open %s: %s\n", file, err.Error())
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		log.Printf("ERROR: NewReader %s: %s\n", file, err.Error())
	}
	defer gz.Close()

	data, err := ioutil.ReadAll(gz)
	if err != nil {
		log.Printf("ERROR: ReadAll: %v\n", err.Error())
		return nil
	}
	return data
}

// getDocument returns goquery document
func getDocument(uri string) (*goquery.Document, error) {
	res, err := getResponse(uri)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.New("http.Response is nil")
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Printf("ERROR: NewDocumentFromResponse %s: %s\n", uri, err.Error())
		return nil, err
	}

	if doc == nil {
		return nil, errors.New("Document is nil")
	}

	return doc, nil
}

// getClient returns http client
func getClient(torProxy bool) (*http.Client, error) {
	if torProxy && ttor.Running() {
		proxyURL, err := url.Parse("socks5://127.0.0.1:" + ttor.Port)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse proxy URL: %v\n", err)
		}

		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("Failed to obtain proxy dialer: %v\n", err)
		}

		dc := dialer.(interface {
			DialContext(ctx context.Context, network, addr string) (net.Conn, error)
		})

		return &http.Client{
			Transport: &http.Transport{
				DialContext:     dc.DialContext,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 30 * time.Second,
		}, nil
	} else {
		baseDialer := &net.Dialer{
			Timeout: 30 * time.Second,
		}
		dialContext := (baseDialer).DialContext

		return &http.Client{
			Transport: &http.Transport{
				DialContext:     dialContext,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 30 * time.Second,
		}, nil
	}
}

// getResponse returns http response
func getResponse(uri string) (*http.Response, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, errors.New(strings.Replace(err.Error(), tmdbApiKey, "xxx", -1))
	}

	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:36.0) Gecko/20100101 Firefox/36.0")

	client, err := getClient(strings.Contains(uri, ".onion/"))
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, errors.New(strings.Replace(err.Error(), tmdbApiKey, "xxx", -1))
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusTooManyRequests {
		return nil, errors.New(fmt.Sprintf("Status Code %d received", res.StatusCode))
	}

	return res, nil
}

// getBody returns response body as byte slice
func getBody(uri string) ([]byte, error) {
	var err error
	var body []byte
	var res *http.Response

	retry := func(r *http.Response) (*http.Response, error) {
		sleep, err := strconv.Atoi(r.Header.Get("Retry-After"))
		if err == nil {
			if verbose {
				log.Printf("TMDB: Retry-After: %d, sleeping for %d seconds\n", sleep, sleep+1)
			}

			select {
			case <-cancelchan:
				return nil, errors.New("getBody request canceled")
			case <-time.After(time.Duration(sleep+1) * time.Second):
				break
			}

			return getResponse(uri)
		}

		return r, nil
	}

	res, err = getResponse(uri)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusTooManyRequests {
		res, err = retry(res)
		if err != nil {
			return nil, err
		}

		if res.StatusCode == http.StatusTooManyRequests {
			res, err = retry(res)
			if err != nil {
				return nil, err
			}

			if res.StatusCode == http.StatusTooManyRequests {
				res, err = retry(res)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("StatusCode %d received", res.StatusCode)
	}

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	res.Body.Close()

	return body, nil
}

// getTitle returns title from torrent title
func getTitle(torrentTitle string) string {
	title := strings.ToLower(torrentTitle)
	title = strings.TrimSuffix(title, "\n")
	title = strings.Replace(title, ".", " ", -1)
	title = strings.Replace(title, "-", " ", -1)
	title = strings.Replace(title, ":", "", -1)

	reEp := reSeason.FindAllStringSubmatch(title, -1)
	if len(reEp) > 1 {
		title = reSeason.ReplaceAllString(title, " ")
	}

	re1 := reTitle1.FindAllStringSubmatch(title, -1)
	if len(re1) > 0 {
		title = re1[0][1]
	}

	re2 := reTitle2.FindAllStringSubmatch(title, -1)
	if len(re2) > 0 {
		title = re2[0][1]
	}

	title = strings.Replace(title, "(", "", -1)
	title = strings.Replace(title, ")", "", -1)

	title = reSeason.ReplaceAllString(title, "")

	return strings.Trim(title, " ")
}

// getYear returns year from torrent title
func getYear(torrentTitle string) string {
	year := ""
	re := reYear.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		year = re[0][2]
	}

	return year
}

// getQuality returns quality from torrent title
func getQuality(torrentTitle string) string {
	quality := ""
	re := reQuality.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		quality = re[0][2]
	}

	return quality
}

// getSeason returns tv show season from torrent title
func getSeason(torrentTitle string) string {
	season := ""
	re := reSeason.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		season = re[0][1]
	}

	return season
}

// getEpisode returns tv show episode from torrent title
func getEpisode(torrentTitle string) string {
	episode := ""
	re := reSeason.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		episode = re[0][2]
	}

	return episode
}

// getCast returns cast string from []tmdbCast
func getCast(res []tmdbCast) []string {
	var cast []string
	for _, c := range res {
		cast = append(cast, c.Name)
	}

	return cast
}

// getCastIds returns cast ids from []tmdbCast
func getCastIds(res []tmdbCast) []int {
	var cast []int
	for _, c := range res {
		cast = append(cast, c.Id)
	}

	return cast
}

// getVideo returns youtube trailer from tmdb []tmdbVideo
func getVideo(res []tmdbVideo) (video string) {
	for _, c := range res {
		if strings.ToLower(c.Site) == "youtube" && strings.ToLower(c.Type) == "trailer" {
			video = c.Key
			return
		}
	}

	return
}

// getGenre returns list of genres from []tmdbGenre
func getGenre(res []tmdbGenre) []string {
	var genre []string
	for _, g := range res {
		genre = append(genre, g.Name)
	}

	return genre
}

// getDirector returns director string from []tmdbCrew
func getDirector(res []tmdbCrew) string {
	var director string
	for _, c := range res {
		if strings.ToLower(c.Job) == "director" {
			return c.Name
		}
	}

	return director
}

// getDirectorId returns cast ids from []tmdbCast
func getDirectorId(res []tmdbCrew) int {
	var director int
	for _, c := range res {
		if strings.ToLower(c.Job) == "director" {
			return c.Id
		}
	}

	return director
}

// getLanguage returns language from string
func getLanguage(name string) *language {
	lang := new(language)
	for _, l := range languages {
		if strings.ToLower(name) == strings.ToLower(l.Name) {
			lang = &l
			break
		}
	}

	return lang
}

// getTpbHost returns tpb host
func getTpbHost() string {
	if ttor.Running() {
		return TpbTor
	}

	for _, host := range TpbHosts {
		_, err := net.DialTimeout("tcp", host+":80", time.Duration(3)*time.Second)
		if err == nil {
			if verbose {
				log.Printf("TPB: Using host %s\n", host)
			}
			return host
		}
	}

	if verbose {
		log.Printf("TPB: Using first host %s\n", TpbHosts[0])
	}

	return TpbHosts[0]
}

// getEztvHost returns eztv host
func getEztvHost() string {
	for _, host := range EztvHosts {
		_, err := net.DialTimeout("tcp", host+":80", time.Duration(3)*time.Second)
		if err == nil {
			if verbose {
				log.Printf("EZTV: Using host %s\n", host)
			}
			return host
		}
	}

	if verbose {
		log.Printf("EZTV: Using first host %s\n", EztvHosts[0])
	}

	return EztvHosts[0]
}
