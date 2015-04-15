package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	tmdb "github.com/amahi/go-themoviedb"
	humanize "github.com/dustin/go-humanize"
	"github.com/xrash/smetrics"
)

var (
	appName    = "bukanir-http"
	appVersion = "1.3"
)

type torrent struct {
	Title          string
	FormattedTitle string
	MagnetLink     string
	Year           string
	Size           uint64
	SizeHuman      string
	Seeders        int
	Category       int
	Season         int
	Episode        int
}

type movie struct {
	Id           int    `json:"id"`
	Title        string `json:"title"`
	Year         string `json:"year"`
	PosterSmall  string `json:"posterSmall"`
	PosterMedium string `json:"posterMedium"`
	PosterLarge  string `json:"posterLarge"`
	PosterXLarge string `json:"posterXLarge"`
	Size         uint64 `json:"size"`
	SizeHuman    string `json:"sizeHuman"`
	Seeders      int    `json:"seeders"`
	MagnetLink   string `json:"magnetLink"`
	Release      string `json:"release"`
	Category     int    `json:"category"`
	Season       int    `json:"season"`
	Episode      int    `json:"episode"`
}

type summary struct {
	Id       int     `json:"id"`
	Cast     string  `json:"cast"`
	Rating   float64 `json:"rating"`
	TagLine  string  `json:"tagline"`
	Overview string  `json:"overview"`
	Runtime  int     `json:"runtime"`
}

type subtitle struct {
	Id           string  `json:"id"`
	Title        string  `json:"title"`
	Year         string  `json:"year"`
	Release      string  `json:"release"`
	DownloadLink string  `json:"downloadLink"`
	Score        float64 `json:"score"`
}

type bySeeders []movie

func (a bySeeders) Len() int           { return len(a) }
func (a bySeeders) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a bySeeders) Less(i, j int) bool { return a[i].Seeders > a[j].Seeders }

type byScore []subtitle

func (a byScore) Len() int           { return len(a) }
func (a byScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byScore) Less(i, j int) bool { return a[i].Score > a[j].Score }

const tmdbApiKey = "YOUR_API_KEY"

var (
	reHTML   = regexp.MustCompile(`<[^>]*>`)
	reYear   = regexp.MustCompile(`(.*)(19\d{2}|20\d{2})(.*)`)
	reTitle1 = regexp.MustCompile(`(.*?)(dvdrip|xvid|dvdscr|brrip|bdrip|divx|klaxxon|hc|webrip|hdrip|camrip|hdtv|eztv|proper|x264|480p|720p|1080p|[\*\{\(\[]?[0-9]{4}).*`)
	reTitle2 = regexp.MustCompile(`(.*?)\(.*\)(.*)`)
	reSeason = regexp.MustCompile(`(?i:s|season)(\d{2})(?i:e|x|episode)(\d{2})`)
)

var categories = []string{
	"201",
	"207",
	"205",
}

var hosts = []string{
	"thepiratebay.se",
	"thepiratebay.mk",
	"thepiratebay.cd",
	"thepiratebay.lv",
}

var trackers = []string{
	"udp://tracker.publicbt.com:80/announce",
	"udp://tracker.openbittorrent.com:80/announce",
	"udp://open.demonii.com:1337/announce",
	"udp://tracker.istole.it:6969",
	"udp://tracker.coppersurfer.tk:80",
}

var movies []movie
var torrents []torrent
var subtitles []subtitle
var movieSummary summary
var wg sync.WaitGroup

var chain = `-----BEGIN CERTIFICATE-----
MIIFRzCCBC+gAwIBAgISESGXNII/8fVUAIsyFQbH5pmTMA0GCSqGSIb3DQEBBQUA
MF0xCzAJBgNVBAYTAkJFMRkwFwYDVQQKExBHbG9iYWxTaWduIG52LXNhMTMwMQYD
VQQDEypHbG9iYWxTaWduIE9yZ2FuaXphdGlvbiBWYWxpZGF0aW9uIENBIC0gRzIw
HhcNMTQxMDExMTAwODE1WhcNMTUxMDEyMTAwODE1WjBuMQswCQYDVQQGEwJVUzEL
MAkGA1UECBMCQ0ExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xGTAXBgNVBAoTEENs
b3VkRmxhcmUsIEluYy4xHzAdBgNVBAMTFnNzbDIwMDAuY2xvdWRmbGFyZS5jb20w
ggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCsrcZa6V46wy4fFmaLfHT1
FPZXa+2qDvaWpAbrT5N9ibIuLnJ0yvpkB5WvI3R21pXOeqqAZsSKTeukUIQV2fTD
yxynffWFPdSsj5kRu5tR0JvDS4fQz5NZaxYemhjPSe7mV+fvNk0aWW8xBbU1jJfr
i/84f62sgcjQJy0SV+6/sXZzcQt6DM4MPn1aNWWritxNDwx8l0T38Zj0x6Zcy4tV
2B4kfDMA9+8kuC+HaSkzNIQKt2uN5xrUivs7nA6H6oZlV7YDOm6nhC2OtOmjXQKH
Gga4Nq9+UefCQ9rLL9cylYWvO0BqHh0nLbxaCEOlo+362E6TDptm/qoN+mg083RH
AgMBAAGjggHuMIIB6jAOBgNVHQ8BAf8EBAMCBaAwSQYDVR0gBEIwQDA+BgZngQwB
AgIwNDAyBggrBgEFBQcCARYmaHR0cHM6Ly93d3cuZ2xvYmFsc2lnbi5jb20vcmVw
b3NpdG9yeS8wQwYDVR0RBDwwOoIWc3NsMjAwMC5jbG91ZGZsYXJlLmNvbYIOY2xv
dWRmbGFyZS5jb22CECouY2xvdWRmbGFyZS5jb20wCQYDVR0TBAIwADAdBgNVHSUE
FjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwRQYDVR0fBD4wPDA6oDigNoY0aHR0cDov
L2NybC5nbG9iYWxzaWduLmNvbS9ncy9nc29yZ2FuaXphdGlvbnZhbGcyLmNybDCB
lgYIKwYBBQUHAQEEgYkwgYYwRwYIKwYBBQUHMAKGO2h0dHA6Ly9zZWN1cmUuZ2xv
YmFsc2lnbi5jb20vY2FjZXJ0L2dzb3JnYW5pemF0aW9udmFsZzIuY3J0MDsGCCsG
AQUFBzABhi9odHRwOi8vb2NzcDIuZ2xvYmFsc2lnbi5jb20vZ3Nvcmdhbml6YXRp
b252YWxnMjAdBgNVHQ4EFgQUH6U3xLIIaPv8vcp1Zzi6jFtNa94wHwYDVR0jBBgw
FoAUXUayjcRLdBy77fVztjq3OI91nn4wDQYJKoZIhvcNAQEFBQADggEBAEviNeXx
Qv6zHbRs/AhmbtdJDaiNZVe6RF20CnPev+X4H8XVwha80GgNqdUCBIuQZIJ+L7lB
NMxAAp+XuCW/4F959ZQtAsZkiFaMUf7NI7Bpl61W15aQPVplt18EkMpCf3CBXFCq
J8R/oJilzJRdh0bQ2yIL6IDIG/bCZ9GXh9TKBKJC6MUzsf1GMziihytg/510dng0
Nwp1/q+0XioOsxpOp3qX2LnC/datjsEIHtjIr8LnZZojh3RG2cuMTS3n5fiwxXp2
9Gg/FqkTXHfWBgdzZ7wD8NAPxak03AlDjQthXEn4YwB/c8CjKqn+r77o4pvLm+JE
bJLtKDaYbNmULxY=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIEYDCCA0igAwIBAgILBAAAAAABL07hRQwwDQYJKoZIhvcNAQEFBQAwVzELMAkG
A1UEBhMCQkUxGTAXBgNVBAoTEEdsb2JhbFNpZ24gbnYtc2ExEDAOBgNVBAsTB1Jv
b3QgQ0ExGzAZBgNVBAMTEkdsb2JhbFNpZ24gUm9vdCBDQTAeFw0xMTA0MTMxMDAw
MDBaFw0yMjA0MTMxMDAwMDBaMF0xCzAJBgNVBAYTAkJFMRkwFwYDVQQKExBHbG9i
YWxTaWduIG52LXNhMTMwMQYDVQQDEypHbG9iYWxTaWduIE9yZ2FuaXphdGlvbiBW
YWxpZGF0aW9uIENBIC0gRzIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIB
AQDdNR3yIFQmGtDvpW+Bdllw3Of01AMkHyQOnSKf1Ccyeit87ovjYWI4F6+0S3qf
ZyEcLZVUunm6tsTyDSF0F2d04rFkCJlgePtnwkv3J41vNnbPMYzl8QbX3FcOW6zu
zi2rqqlwLwKGyLHQCAeV6irs0Z7kNlw7pja1Q4ur944+ABv/hVlrYgGNguhKujiz
4MP0bRmn6gXdhGfCZsckAnNate6kGdn8AM62pI3ffr1fsjqdhDFPyGMM5NgNUqN+
ARvUZ6UYKOsBp4I82Y4d5UcNuotZFKMfH0vq4idGhs6dOcRmQafiFSNrVkfB7cVT
5NSAH2v6gEaYsgmmD5W+ZoiTAgMBAAGjggElMIIBITAOBgNVHQ8BAf8EBAMCAQYw
EgYDVR0TAQH/BAgwBgEB/wIBADAdBgNVHQ4EFgQUXUayjcRLdBy77fVztjq3OI91
nn4wRwYDVR0gBEAwPjA8BgRVHSAAMDQwMgYIKwYBBQUHAgEWJmh0dHBzOi8vd3d3
Lmdsb2JhbHNpZ24uY29tL3JlcG9zaXRvcnkvMDMGA1UdHwQsMCowKKAmoCSGImh0
dHA6Ly9jcmwuZ2xvYmFsc2lnbi5uZXQvcm9vdC5jcmwwPQYIKwYBBQUHAQEEMTAv
MC0GCCsGAQUFBzABhiFodHRwOi8vb2NzcC5nbG9iYWxzaWduLmNvbS9yb290cjEw
HwYDVR0jBBgwFoAUYHtmGkUNl8qJUC99BM00qP/8/UswDQYJKoZIhvcNAQEFBQAD
ggEBABvgiADHBREc/6stSEJSzSBo53xBjcEnxSxZZ6CaNduzUKcbYumlO/q2IQen
fPMOK25+Lk2TnLryhj5jiBDYW2FQEtuHrhm70t8ylgCoXtwtI7yw07VKoI5lkS/Z
9oL2dLLffCbvGSuXL+Ch7rkXIkg/pfcNYNUNUUflWP63n41edTzGQfDPgVRJEcYX
pOBWYdw9P91nbHZF2krqrhqkYE/Ho9aqp9nNgSvBZnWygI/1h01fwlr1kMbawb30
hag8IyrhFHvBN91i0ZJsumB9iOQct+R2UTjEqUdOqCsukNK1OFHrwZyKarXMsh3o
wFZUTKiL8IkyhtyTMr5NGvo1dbU=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIFZDCCBQmgAwIBAgIRAIxVL2C5eOVJT5AmeRKBrGIwCgYIKoZIzj0EAwIwgZIx
CzAJBgNVBAYTAkdCMRswGQYDVQQIExJHcmVhdGVyIE1hbmNoZXN0ZXIxEDAOBgNV
BAcTB1NhbGZvcmQxGjAYBgNVBAoTEUNPTU9ETyBDQSBMaW1pdGVkMTgwNgYDVQQD
Ey9DT01PRE8gRUNDIERvbWFpbiBWYWxpZGF0aW9uIFNlY3VyZSBTZXJ2ZXIgQ0Eg
MjAeFw0xNTA0MDEwMDAwMDBaFw0xNTA5MzAyMzU5NTlaMGsxITAfBgNVBAsTGERv
bWFpbiBDb250cm9sIFZhbGlkYXRlZDEhMB8GA1UECxMYUG9zaXRpdmVTU0wgTXVs
dGktRG9tYWluMSMwIQYDVQQDExpzbmkzMzc4MC5jbG91ZGZsYXJlc3NsLmNvbTBZ
MBMGByqGSM49AgEGCCqGSM49AwEHA0IABCOCgOk1v/kBW5YJQlP3ZdXKEv84hJDs
DIEHlt3yKkEKjG0u0XCL6emo8VPC6hq7Sk0YTiiFhZuiveYjvAnPSLCjggNkMIID
YDAfBgNVHSMEGDAWgBRACWFn8LyDcU/eEggsb9TUK3Y9ljAdBgNVHQ4EFgQUAsKS
prtyTgwn073RV3zrgvLwmYwwDgYDVR0PAQH/BAQDAgeAMAwGA1UdEwEB/wQCMAAw
HQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCME8GA1UdIARIMEYwOgYLKwYB
BAGyMQECAgcwKzApBggrBgEFBQcCARYdaHR0cHM6Ly9zZWN1cmUuY29tb2RvLmNv
bS9DUFMwCAYGZ4EMAQIBMFYGA1UdHwRPME0wS6BJoEeGRWh0dHA6Ly9jcmwuY29t
b2RvY2E0LmNvbS9DT01PRE9FQ0NEb21haW5WYWxpZGF0aW9uU2VjdXJlU2VydmVy
Q0EyLmNybDCBiAYIKwYBBQUHAQEEfDB6MFEGCCsGAQUFBzAChkVodHRwOi8vY3J0
LmNvbW9kb2NhNC5jb20vQ09NT0RPRUNDRG9tYWluVmFsaWRhdGlvblNlY3VyZVNl
cnZlckNBMi5jcnQwJQYIKwYBBQUHMAGGGWh0dHA6Ly9vY3NwLmNvbW9kb2NhNC5j
b20wggGrBgNVHREEggGiMIIBnoIac25pMzM3ODAuY2xvdWRmbGFyZXNzbC5jb22C
ECouYWNyZXN0cnVzdC5jb22CDCouYWR1d2F0YS5sa4INKi5iZWVlZWVyLm9yZ4IV
Ki5rbmlnaHRhc2NlbnNpb24uY29tgg4qLnB1bmRyb2lkLmNvbYIOKi50ZXJyYWlj
dC5jb22CESoudGhlcGlyYXRlYmF5LnNlghUqLnRyYXZlbHdvbmRlcjM2NS5jb22C
CCoudWNlLnB3gg8qLnVwbG9hZGJheS5vcmeCHyoueG4tLTEyY2FhNmhnMWEzYTJi
NWQwZDljdmgud3OCDmFjcmVzdHJ1c3QuY29tggphZHV3YXRhLmxrggtiZWVlZWVy
Lm9yZ4ITa25pZ2h0YXNjZW5zaW9uLmNvbYIMcHVuZHJvaWQuY29tggx0ZXJyYWlj
dC5jb22CD3RoZXBpcmF0ZWJheS5zZYITdHJhdmVsd29uZGVyMzY1LmNvbYIGdWNl
LnB3gg11cGxvYWRiYXkub3Jngh14bi0tMTJjYWE2aGcxYTNhMmI1ZDBkOWN2aC53
czAKBggqhkjOPQQDAgNJADBGAiEAxfsGDGwsOfSYaLdeVunpv5Exjy010KZp9l1+
9Yd9eDICIQDjZ4weh631w+0AnIZF4crePccCxsZpSFphWmUA7kpRmg==
-----END CERTIFICATE-----`

func decodePem(certInput string) tls.Certificate {
	var cert tls.Certificate
	certPEMBlock := []byte(certInput)
	var certDERBlock *pem.Block
	for {
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
		}
	}
	return cert
}

func getTLSConfig() tls.Config {
	certChain := decodePem(chain)
	conf := tls.Config{}
	conf.RootCAs = x509.NewCertPool()
	for _, cert := range certChain.Certificate {
		x509Cert, err := x509.ParseCertificate(cert)
		if err != nil {
			panic(err)
		}
		conf.RootCAs.AddCert(x509Cert)
	}
	conf.BuildNameToCertificate()
	return conf
}

func tpbTop(category string) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tpbTop: ", r)
		}
	}()

	host := getHost()
	uri := "https://%s/top/%s"

	doc, err := getDocument(fmt.Sprintf(uri, host, category))
	if err != nil {
		log.Print("Error making TPB call: %v", err.Error())
		return
	}

	getTorrents(doc)
}

func tpbSearch(query string, page int) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tpbSearch: ", r)
		}
	}()

	host := getHost()
	uri := "https://%s/search/%s/%d/7/201,207,205"

	doc, err := getDocument(fmt.Sprintf(uri, host, url.QueryEscape(query), page))
	if err != nil {
		log.Print("Error making TPB call: %v", err.Error())
		return
	}

	getTorrents(doc)
}

func tmdbSearch(t torrent) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tmdbSearch: ", r)
		}
	}()

	md := tmdb.Init(tmdbApiKey)
	config, _ := md.GetConfig()

	var results tmdb.TmdbResponse
	if t.Category == 205 {
		results, _ = md.SearchTmdbtv(t.FormattedTitle)
	} else {
		results, _ = md.SearchMovie(t.FormattedTitle)
	}

	if results.Total_results == 0 {
		return
	}

	var res *tmdb.TmdbResult
	if t.Category == 205 {
		res = &results.Results[0]
	} else {
		res = new(tmdb.TmdbResult)
		for _, result := range results.Results {
			if result.Release_date != "" && t.Year != "" {
				tmdbYear, _ := strconv.Atoi(getYear(result.Release_date))
				torrentYear, _ := strconv.Atoi(t.Year)
				if tmdbYear == torrentYear || tmdbYear == torrentYear-1 || tmdbYear == torrentYear+1 {
					res = &result
					break
				}
			}
		}
	}

	if res.Id == 0 {
		return
	}

	var p tmdb.TmdbPoster
	if t.Category == 205 {
		p, _ = md.GetTmdbTvImages(strconv.Itoa(res.Id), t.Season)
	}

	var posterSmall, posterMedium, posterLarge, posterXLarge string
	if t.Category == 205 && len(p.Posters) > 0 {
		posterSmall = config.Images.Base_url + config.Images.Poster_sizes[0] + p.Posters[0].File_path
		posterMedium = config.Images.Base_url + config.Images.Poster_sizes[3] + p.Posters[0].File_path
		posterLarge = config.Images.Base_url + config.Images.Poster_sizes[3] + p.Posters[0].File_path
		posterXLarge = config.Images.Base_url + config.Images.Poster_sizes[4] + p.Posters[0].File_path
	} else {
		posterSmall = config.Images.Base_url + config.Images.Poster_sizes[0] + res.Poster_path
		posterMedium = config.Images.Base_url + config.Images.Poster_sizes[3] + res.Poster_path
		posterLarge = config.Images.Base_url + config.Images.Poster_sizes[3] + res.Poster_path
		posterXLarge = config.Images.Base_url + config.Images.Poster_sizes[4] + res.Poster_path
	}

	size := len(config.Images.Poster_sizes)
	if size < 5 {
		return
	}

	var title string
	var year string
	if t.Category == 205 {
		title = res.Original_name
		year = getYear(res.First_air_date)
	} else {
		title = res.Title
		year = getYear(res.Release_date)
	}

	m := movie{
		res.Id,
		title,
		year,
		posterSmall,
		posterMedium,
		posterLarge,
		posterXLarge,
		t.Size,
		t.SizeHuman,
		t.Seeders,
		t.MagnetLink,
		t.Title,
		t.Category,
		t.Season,
		t.Episode,
	}
	movies = append(movies, m)
}

func tmdbSummary(id int, category int, season int) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in tmdbSummary: ", r)
		}
	}()

	md := tmdb.Init(tmdbApiKey)

	var err error
	var res tmdb.MovieMetadata

	if category == 205 {
		res, err = md.GetTmdbTvDetails(strconv.Itoa(id))
	} else {
		res, err = md.GetMovieDetails(strconv.Itoa(id))
	}
	if err != nil {
		return
	}

	if category == 205 {
		res.Credits, err = md.GetTmdbTvCredits(strconv.Itoa(id), season)
	} else {
		res.Credits, err = md.GetMovieCredits(strconv.Itoa(id))
	}
	if err != nil {
		return
	}

	res.Config, err = md.GetConfig()
	if err != nil {
		return
	}

	movieSummary = summary{
		id,
		getCast(res.Credits.Cast),
		res.Vote_average,
		res.Tagline,
		res.Overview,
		res.Runtime,
	}
}

func podnapisi(movie string, year string, torrentRelease string, lang string, category int, season int, episode int) {
	type Page struct {
		Current int `xml:"current"`
		Count   int `xml:"count"`
		Results int `xml:"results"`
	}

	type Subtitle struct {
		XMLName   xml.Name `xml:"subtitle"`
		Id        int      `xml:"id"`
		Pid       string   `xml:"pid"`
		Title     string   `xml:"title"`
		Year      string   `xml:"year"`
		Url       string   `xml:"url"`
		Release   string   `xml:"release"`
		TvSeason  int      `xml:"tvSeason"`
		TvEpisode int      `xml:"tvEpisode"`
	}

	type Data struct {
		XMLName      xml.Name   `xml:"results"`
		Pagination   Page       `xml:"pagination"`
		SubtitleList []Subtitle `xml:"subtitle"`
	}

	baseUrl := "http://podnapisi.net/subtitles/"
	searchUrl := baseUrl + "search/old?sK=%s&sY=%s&sJ=%s"

	url := fmt.Sprintf(searchUrl, url.QueryEscape(movie), year, lang)

	if category == 205 {
		if season != 0 {
			url = url + fmt.Sprintf("&sTS=%d&sTE=%d", season, episode)
		}
	}

	res, err := httpGet(url)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	var r Data
	err = xml.Unmarshal(body, &r)
	if err != nil {
		log.Printf("Error unmarshalling response: %v\n", err)
	}

	for _, s := range r.SubtitleList {
		rel := strings.Fields(s.Release)
		var subtitleRelease string
		if len(rel) > 0 {
			subtitleRelease = rel[0]
		} else {
			continue
		}

		score := compareRelease(torrentRelease, subtitleRelease)
		if score < 0.4 {
			continue
		}

		if category == 205 {
			if season != s.TvSeason || episode != s.TvEpisode {
				continue
			}
		}

		downloadLink := fmt.Sprintf(baseUrl+"%s/download", s.Pid)

		s := subtitle{strconv.Itoa(s.Id), s.Title, s.Year, subtitleRelease, downloadLink, score}
		subtitles = append(subtitles, s)
	}
}

func titlovi(movie string, torrentRelease string, category int, season int, episode int) {
	searchUrl := "http://en.titlovi.com/subtitles/subtitles.aspx?subtitle=%s"
	downloadUrl := "http://titlovi.com/downloads/default.ashx?type=1&mediaid=%s"

	url := fmt.Sprintf(searchUrl, url.QueryEscape(movie))

	doc, err := getDocument(url)
	if err != nil {
		log.Fatal(err)
	}

	nodes := doc.Find(`li.listing`)
	divs := nodes.Find(`div.title.c1`)

	if divs.Length() == 0 {
		return
	}

	reNum := regexp.MustCompile(`[^0-9]`)

	parseHTML := func(i int, div *goquery.Selection) {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Print("Recovered in parseHTML: ", r)
			}
		}()

		link := div.Find(`a`).First()
		href, _ := link.Attr("href")

		title := getTitle(strings.TrimSpace(link.Text()))

		subtitleYear := div.Find(`span.year`).First().Text()
		year := getYear(subtitleYear)

		subtitleRelease := div.Find(`span.release`).First().Text()

		split := strings.Split(href, "-")
		id := split[len(split)-1]
		id = reNum.ReplaceAllString(id, "")

		downloadLink := fmt.Sprintf(downloadUrl, id)

		score := compareRelease(torrentRelease, subtitleRelease)
		if score < 0.4 {
			return
		}

		if category == 205 {
			rs, _ := strconv.Atoi(getSeason(subtitleRelease))
			re, _ := strconv.Atoi(getEpisode(subtitleRelease))
			if season != rs || episode != re {
				return
			}
		}

		s := subtitle{id, title, year, subtitleRelease, downloadLink, score}
		subtitles = append(subtitles, s)
	}

	wg.Add(divs.Length())
	divs.Each(func(i int, s *goquery.Selection) {
		go parseHTML(i, s)
	})
	wg.Wait()
}

func compareRelease(torrentRelease string, subtitleRelease string) float64 {
	torrentRelease = strings.Replace(torrentRelease, ".", " ", -1)
	torrentRelease = strings.Replace(torrentRelease, "-", " ", -1)
	subtitleRelease = strings.Replace(subtitleRelease, ".", " ", -1)
	subtitleRelease = strings.Replace(subtitleRelease, "-", " ", -1)
	return smetrics.Jaro(torrentRelease, subtitleRelease)
}

func httpGet(uri string) (*http.Response, error) {
	jar, _ := cookiejar.New(nil)
	timeout := time.Duration(30 * time.Second)

	dialTimeout := func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, timeout)
	}

	//tlsConfig := getTLSConfig()

	transport := http.Transport{
		Dial: dialTimeout,
		//TLSClientConfig: &tlsConfig,
	}

	httpClient := http.Client{
		Jar:       jar,
		Transport: &transport,
	}

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:36.0) Gecko/20100101 Firefox/36.0")

	res, err := httpClient.Do(req)
	if err != nil || res == nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, nil
	}

	return res, nil
}

func getDocument(uri string) (*goquery.Document, error) {
	res, err := httpGet(uri)
	if err != nil {
		log.Printf("Error httpGet %s: %v", uri, err.Error())
		return nil, err
	}

	if res == nil {
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Printf("Error NewDocumentFromResponse %s: %v", uri, err.Error())
		return nil, err
	}

	if doc == nil {
		return nil, nil
	}

	return doc, nil
}

func getTorrents(doc *goquery.Document) {
	divs := doc.Find(`div.detName`)

	if divs.Length() == 0 {
		return
	}

	var wgp sync.WaitGroup

	parseHTML := func(i int, s *goquery.Selection) {
		defer wgp.Done()

		parent := s.Parent()
		prev := parent.Prev().First()

		title := s.Find(`a.detLink`).Text()
		magnet, _ := parent.Find(`a[title="Download this torrent using magnet"]`).Attr(`href`)
		desc := parent.Find(`font.detDesc`).Text()
		seeders, _ := strconv.Atoi(parent.Next().Text())

		c, _ := prev.Find(`a[title="More from this category"]`).Last().Attr(`href`)
		category, _ := strconv.Atoi(strings.Replace(c, "/browse/", "", -1))

		if seeders == 0 {
			return
		}

		var size uint64
		var sizeHuman string
		parts := strings.Split(desc, ", ")
		if len(parts) > 1 {
			size, _ = humanize.ParseBytes(strings.Split(parts[1], " ")[1])
			sizeHuman = humanize.IBytes(size)
		}

		season, _ := strconv.Atoi(getSeason(title))
		episode, _ := strconv.Atoi(getEpisode(title))

		t := torrent{
			title,
			getTitle(title),
			boostMagnet(magnet),
			getYear(title),
			size,
			sizeHuman,
			seeders,
			category,
			season,
			episode,
		}

		torrents = append(torrents, t)
	}

	wgp.Add(divs.Length())
	divs.Each(func(i int, s *goquery.Selection) {
		go parseHTML(i, s)
	})
	wgp.Wait()
}

func getTitle(torrentTitle string) string {
	title := strings.ToLower(torrentTitle)
	title = strings.Replace(title, ".", " ", -1)
	title = strings.Replace(title, "-", "", -1)

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
	title := ""
	re := reYear.FindAllStringSubmatch(torrentTitle, -1)
	if len(re) > 0 {
		title = re[0][2]
	}
	return title
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

func getHost() string {
	for _, host := range hosts {
		_, err := net.Dial("tcp", host+":80")
		if err == nil {
			return host
		}
	}
	return hosts[0]
}

func getCast(res []tmdb.TmdbCast) string {
	cast := ""
	castLen := len(res)
	if castLen >= 4 {
		for n, c := range res[0:3] {
			cast += c.Name
			if n != 2 {
				cast += ", "
			} else {
				cast += "..."
			}
		}
	} else if castLen == 3 {
		for n, c := range res[0:2] {
			cast += c.Name
			if n != 2 {
				cast += ", "
			}
		}
	} else if castLen == 2 {
		cast += res[0].Name + ", "
		cast += res[1].Name
	} else {
		cast += res[0].Name
	}
	return cast
}

func isValidCategory(category string) bool {
	for _, cat := range categories {
		if cat == category {
			return true
		}
	}
	return false
}

func boostMagnet(magnet string) string {
	for _, tracker := range trackers {
		magnet += "&tr=" + url.QueryEscape(tracker)
	}
	return magnet
}

func saveCache(key string, data []byte, tmpDir string) {
	file := filepath.Join(tmpDir, key+".json")
	err := ioutil.WriteFile(file, data, 0644)
	if err != nil {
		log.Print("Error writing cache file: %v", err.Error())
	}
}

func getCache(key string, tmpDir string) []byte {
	file := filepath.Join(tmpDir, key+".json")
	info, err := os.Stat(file)
	if err != nil {
		return nil
	}
	mtime := info.ModTime().Unix()
	if time.Now().Unix()-mtime > 43200 {
		return nil
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Print("Error reading cache file: %v", err.Error())
		return nil
	}
	return data
}

func Category(category string, limit int, force int, tmpDir string) (string, error) {
	if force != 1 {
		cache := getCache(category, tmpDir)
		if cache != nil {
			return string(cache[:]), nil
		}
	}

	torrents = make([]torrent, 0)

	wg.Add(1)
	go tpbTop(category)
	wg.Wait()

	if limit > 0 {
		if limit > len(torrents) {
			limit = len(torrents)
		}
		torrents = torrents[0:limit]
	}

	movies = make([]movie, 0)
	wg.Add(len(torrents))
	for _, torrent := range torrents {
		go tmdbSearch(torrent)
	}
	wg.Wait()

	sort.Sort(bySeeders(movies))
	js, err := json.MarshalIndent(movies, "", "    ")
	if err != nil {
		return "empty", err
	}

	if len(movies) > 0 {
		saveCache(category, js, tmpDir)
	}

	return string(js[:]), nil
}

func Search(query string, limit int) (string, error) {
	torrents = make([]torrent, 0)

	wg.Add(3)
	go tpbSearch(query, 0)
	go tpbSearch(query, 1)
	go tpbSearch(query, 2)
	wg.Wait()

	if limit > 0 {
		if limit > len(torrents) {
			limit = len(torrents)
		}
		torrents = torrents[0:limit]
	}

	movies = make([]movie, 0)
	wg.Add(len(torrents))
	for _, torrent := range torrents {
		go tmdbSearch(torrent)
	}
	wg.Wait()

	sort.Sort(bySeeders(movies))
	js, err := json.MarshalIndent(movies, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func Summary(id int, category int, season int) (string, error) {
	movieSummary = summary{}
	wg.Add(1)
	go tmdbSummary(id, category, season)
	wg.Wait()

	js, err := json.MarshalIndent(movieSummary, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func Subtitle(movie string, year string, release string, language string, category int, season int, episode int) (string, error) {
	subtitles = make([]subtitle, 0)
	podnapisi(movie, year, release, language, category, season, episode)
	if language == "36" || language == "10" || language == "38" {
		titlovi(movie, release, category, season, episode)
	}

	if len(subtitles) == 0 && language != "2" {
		podnapisi(movie, year, release, "2", category, season, episode)
	}

	sort.Sort(byScore(subtitles))

	js, err := json.MarshalIndent(subtitles, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func setServer(w http.ResponseWriter) {
	w.Header().Set("Server", fmt.Sprintf("%s/%s", appName, appVersion))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	setServer(w)
	http.Error(w, "403 Forbidden", http.StatusForbidden)
	return
}

func handleCategory(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	category := r.FormValue("c")
	limit, _ := strconv.Atoi(r.FormValue("l"))
	force, _ := strconv.Atoi(r.FormValue("f"))
	tmpdir := r.FormValue("t")

	if isValidCategory(category) {
		js, err := Category(category, limit, force, tmpdir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(js))
	} else {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	query := r.FormValue("q")
	limit, _ := strconv.Atoi(r.FormValue("l"))

	js, err := Search(query, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(js))
}

func handleSummary(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	id, _ := strconv.Atoi(r.FormValue("i"))
	category, _ := strconv.Atoi(r.FormValue("c"))
	season, _ := strconv.Atoi(r.FormValue("s"))

	js, err := Summary(id, category, season)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(js))
}

func handleSubtitle(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	movie := r.FormValue("m")
	year := r.FormValue("y")
	release := r.FormValue("r")
	language := r.FormValue("l")
	category, _ := strconv.Atoi(r.FormValue("c"))
	season, _ := strconv.Atoi(r.FormValue("s"))
	episode, _ := strconv.Atoi(r.FormValue("e"))

	js, err := Subtitle(movie, year, release, language, category, season, episode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(js))
}

func main() {
	bind := flag.String("bind", ":7314", "Bind address")
	flag.Parse()

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/search", handleSearch)
	http.HandleFunc("/category", handleCategory)
	http.HandleFunc("/summary", handleSummary)
	http.HandleFunc("/subtitle", handleSubtitle)

	l, err := net.Listen("tcp4", *bind)
	if err != nil {
		log.Fatal(err)
	}
	http.Serve(l, nil)
	defer l.Close()
}
