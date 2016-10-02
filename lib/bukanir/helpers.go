package bukanir

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
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

// Gets title from torrent title
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

// Gets year from torrent title
func getYear(torrentTitle string) string {
	year := ""
	re := reYear.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		year = re[0][2]
	}
	return year
}

// Gets quality from torrent title
func getQuality(torrentTitle string) string {
	quality := ""
	re := reQuality.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		quality = re[0][2]
	}
	return quality
}

// Gets tv show season from torrent title
func getSeason(torrentTitle string) string {
	season := ""
	re := reSeason.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		season = re[0][1]
	}
	return season
}

// Gets tv show episode from torrent title
func getEpisode(torrentTitle string) string {
	episode := ""
	re := reSeason.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		episode = re[0][2]
	}
	return episode
}

// Gets cast string from []tmdbCast
func getCast(res []tmdbCast) []string {
	var cast []string
	for _, c := range res {
		cast = append(cast, c.Name)
	}
	return cast
}

// Gets youtube trailer from tmdb []tmdbVideo
func getVideo(res []tmdbVideo) string {
	for _, c := range res {
		if strings.ToLower(c.Site) == "youtube" && strings.ToLower(c.Type) == "trailer" {
			return c.Key
		}
	}
	return ""
}

// Gets list of genres from []tmdbGenre
func getGenre(res []tmdbGenre) []string {
	var genre []string
	for _, g := range res {
		genre = append(genre, g.Name)
	}
	return genre
}

// Gets director string from []tmdbCrew
func getDirector(res []tmdbCrew) string {
	var director string
	for _, c := range res {
		if strings.ToLower(c.Job) == "director" {
			return c.Name
		}
	}
	return director
}

// Gets language from string
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

// Gets tpb host
func getTpbHost() string {
	for _, host := range tpbHosts {
		_, err := net.DialTimeout("tcp", host+":80", time.Duration(3)*time.Second)
		if err == nil {
			if verbose {
				log.Printf("TPB: Using host %s\n", host)
			}
			return host
		}
	}
	if verbose {
		log.Printf("TPB: Using first host %s\n", tpbHosts[0])
	}
	return tpbHosts[0]
}

// Gets eztv host
func getEztvHost() string {
	for _, host := range eztvHosts {
		_, err := net.DialTimeout("tcp", host+":80", time.Duration(3)*time.Second)
		if err == nil {
			if verbose {
				log.Printf("EZTV: Using host %s\n", host)
			}
			return host
		}
	}
	if verbose {
		log.Printf("EZTV: Using first host %s\n", eztvHosts[0])
	}
	return eztvHosts[0]
}

// Gets goquery document
func getDocument(uri string, fast bool) (*goquery.Document, error) {
	res, err := getResponse(uri, fast)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.New("http.Response is nil")
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Printf("ERROR: NewDocumentFromResponse %s: %s\n", uri, err.Error())
		return nil, err
	}

	if doc == nil {
		return nil, errors.New("Document is nil")
	}

	return doc, nil
}

// Gets http response
func getResponse(uri string, fast bool) (*http.Response, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, errors.New(strings.Replace(err.Error(), tmdbApiKey, "xxx", -1))
	}

	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:36.0) Gecko/20100101 Firefox/36.0")

	var res *http.Response

	if fast {
		res, err = clientFast.Do(req)
		if err != nil || res == nil {
			return nil, errors.New(strings.Replace(err.Error(), tmdbApiKey, "xxx", -1))
		}
	} else {
		res, err = clientSlow.Do(req)
		if err != nil || res == nil {
			return nil, errors.New(strings.Replace(err.Error(), tmdbApiKey, "xxx", -1))
		}
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusTooManyRequests {
		return nil, errors.New(fmt.Sprintf("Status Code %d received", res.StatusCode))
	}

	return res, nil
}

// Gets response body as byte
func getBody(uri string) ([]byte, error) {
	var err error
	var body []byte
	var res *http.Response

	// TMDB retry
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

			return getResponse(uri, false)
		}

		return r, nil
	}

	res, err = getResponse(uri, false)
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
		return nil, errors.New(
			fmt.Sprintf("StatusCode %d received", res.StatusCode))
	}

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return body, nil
}
