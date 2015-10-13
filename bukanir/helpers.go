package bukanir

import (
	"compress/gzip"
	"crypto/md5"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	reYear    = regexp.MustCompile(`(.*)(19\d{2}|20\d{2})(.*)`)
	reQuality = regexp.MustCompile(`(.*)(720|1080)p?(.*)`)
	reTitle1  = regexp.MustCompile(`(.*?)(dvdrip|xvid|dvdscr|brrip|bdrip|divx|klaxxon|hc|webrip|hdrip|camrip|hdtv|eztv|proper|x264|480p|720p|1080p|[\*\{\(\[]?[0-9]{4}).*`)
	reTitle2  = regexp.MustCompile(`(.*?)\(.*\)(.*)`)
	reSeason  = regexp.MustCompile(`(?i:s|season)(\d{2})(?i:e|x|episode)(\d{2}).*`)
)

func getTitle(torrentTitle string) string {
	title := strings.ToLower(torrentTitle)
	title = strings.Replace(title, ".", " ", -1)
	title = strings.Replace(title, "-", " ", -1)

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

func getYear(torrentTitle string) string {
	year := ""
	re := reYear.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		year = re[0][2]
	}
	return year
}

func getQuality(torrentTitle string) string {
	quality := ""
	re := reQuality.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		quality = re[0][2]
	}
	return quality
}

func getSeason(torrentTitle string) string {
	season := ""
	re := reSeason.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		season = re[0][1]
	}
	return season
}

func getEpisode(torrentTitle string) string {
	episode := ""
	re := reSeason.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		episode = re[0][2]
	}
	return episode
}

func getCast(res []tmdbCast) []string {
	var cast []string
	for _, c := range res {
		cast = append(cast, c.Name)
	}
	return cast
}

func getVideo(res []video) string {
	for _, c := range res {
		if strings.ToLower(c.Site) == "youtube" && strings.ToLower(c.Type) == "trailer" {
			return c.Key
		}
	}
	return ""
}

func getGenre(res []tmdbGenre) []string {
	var genre []string
	for _, g := range res {
		genre = append(genre, g.Name)
	}
	return genre
}

func getDirector(res []tmdbCrew) string {
	var director string
	for _, c := range res {
		if strings.ToLower(c.Job) == "director" {
			return c.Name
		}
	}
	return director
}

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

func getDocument(uri string) (*goquery.Document, error) {
	res, err := httpGetResponse(uri)
	if err != nil {
		log.Printf("Error httpGetResponse %s: %v\n", uri, err.Error())
		return nil, err
	}

	if verbose {
		log.Printf("Get %s\n", uri)
	}

	if res == nil {
		return nil, errors.New("httpGetResponse is nil")
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Printf("Error NewDocumentFromResponse %s: %v\n", uri, err.Error())
		return nil, err
	}

	if doc == nil {
		return nil, errors.New("getDocument is nil")
	}

	return doc, nil
}

func httpGetResponse(uri string) (*http.Response, error) {
	jar, _ := cookiejar.New(nil)
	timeout := time.Duration(5 * time.Second)

	dialTimeout := func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, timeout)
	}

	transport := http.Transport{
		Dial:            dialTimeout,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	httpClient := http.Client{
		Jar:       jar,
		Transport: &transport,
	}

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Close = true
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:36.0) Gecko/20100101 Firefox/36.0")

	res, err := httpClient.Do(req)
	if err != nil || res == nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("Status Code %d received", res.StatusCode))
	}

	return res, nil
}

func httpGetBody(uri string) ([]byte, error) {
	var err error
	var body []byte
	var res *http.Response

	res, err = http.Get(uri)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 429 {
		sleep, err := strconv.Atoi(res.Header.Get("Retry-After"))
		if err == nil {
			if verbose {
				log.Printf("Retry-After: %d, sleeping for %d seconds\n", sleep, sleep+1)
			}

			time.Sleep(time.Duration(sleep+1) * time.Second)

			res, err = http.Get(uri)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if res.StatusCode != 200 {
		return nil, errors.New(
			fmt.Sprintf("Error httpGetBody: StatusCode %d received", res.StatusCode))
	}

	defer res.Body.Close()
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func saveCache(key string, data []byte, cacheDir string) {
	md5key := md5.Sum([]byte(strings.ToLower(key)))
	file := filepath.Join(cacheDir, fmt.Sprintf("%x.json.gz", md5key))

	err := os.MkdirAll(cacheDir, 0777)
	if err != nil {
		log.Printf("Error creating cache directory %s: %v\n", cacheDir, err.Error())
	}

	f, err := os.Create(file)
	if err != nil {
		log.Printf("Error creating cache file %s: %v\n", file, err.Error())
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	_, err = gz.Write(data)
	if err != nil {
		log.Printf("Error writing gz data: %v\n", err.Error())
	}
}

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
		log.Printf("Using cache file %s\n", file)
	}

	f, err := os.Open(file)
	if err != nil {
		log.Printf("Error opening cache file %s: %v\n", file, err.Error())
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		log.Printf("Error creating gzip reader %s: %v\n", file, err.Error())
	}
	defer gz.Close()

	data, err := ioutil.ReadAll(gz)
	if err != nil {
		log.Printf("Error reading cache file: %v\n", err.Error())
		return nil
	}
	return data
}
