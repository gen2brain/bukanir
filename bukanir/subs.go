package bukanir

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/oz/osdb"
	"github.com/xrash/smetrics"
)

// Language struct
type language struct {
	Name      string
	ISO_639_2 string
	ID        string
}

// Minimum score required
var scoreRequired float64 = 0.65

// Supported languages
var languages = []language{
	{"Albanian", "alb", "29"},
	{"Arabic", "ara", "12"},
	{"Belarus", "bel", "50"},
	{"Bengali", "ben", "59"},
	{"Bosnian", "bos", "10"},
	{"Bulgarian", "bul", "33"},
	{"Catalan", "cat", "53"},
	{"Chinese", "zho", "17"},
	{"Croatian", "hrv", "38"},
	{"Czech", "ces", "7"},
	{"Danish", "dan", "24"},
	{"Dutch", "dut", "23"},
	{"English", "eng", "2"},
	{"Estonian", "est", "20"},
	{"Finnish", "fin", "31"},
	{"French", "fra", "8"},
	{"German", "ger", "5"},
	{"Greek", "gre", "16"},
	{"Hebrew", "heb", "22"},
	{"Hindi", "hin", "42"},
	{"Hungarian", "hun", "15"},
	{"Icelandic", "isl", "6"},
	{"Indonesian", "ind", "54"},
	{"Irish", "gle", "49"},
	{"Italian", "ita", "9"},
	{"Japanese", "jpn", "11"},
	{"Kazakh", "kaz", "58"},
	{"Korean", "kor", "4"},
	{"Latvian", "lav", "21"},
	{"Lithuanian", "lit", "19"},
	{"Macedonian", "mkd", "35"},
	{"Malay", "msa", "55"},
	{"Norwegian", "nor", "3"},
	{"Polish", "pol", "26"},
	{"Portuguese", "por", "32"},
	{"Romanian", "ron", "13"},
	{"Russian", "rus", "27"},
	{"Serbian", "srp", "36"},
	{"Sinhala", "sin", "56"},
	{"Slovak", "slk", "37"},
	{"Slovenian", "slv", "1"},
	{"Spanish", "spa", "28"},
	{"Swedish", "swe", "25"},
	{"Thai", "tha", "44"},
	{"Turkish", "tur", "30"},
	{"Ukrainian", "ukr", "46"},
	{"Vietnamese", "vie", "51"},
}

// podnapisi.net subtitles
func podnapisi(movie string, year string, torrentRelease string, lang string, category int, season int, episode int) {
	defer wgs.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("SUB: Recovered in podnapisi")
		}
	}()

	type Page struct {
		Current int `xml:"current"`
		Count   int `xml:"count"`
		Results int `xml:"results"`
	}

	type Subtitle struct {
		XMLName    xml.Name `xml:"subtitle"`
		Id         int      `xml:"id"`
		Pid        string   `xml:"pid"`
		Title      string   `xml:"title"`
		Year       string   `xml:"year"`
		Url        string   `xml:"url"`
		Release    string   `xml:"release"`
		TvSeason   int      `xml:"tvSeason"`
		TvEpisode  int      `xml:"tvEpisode"`
		Language   string   `xml:"language"`
		LanguageID int      `xml:"languageId"`
	}

	type Data struct {
		XMLName      xml.Name   `xml:"results"`
		Pagination   Page       `xml:"pagination"`
		SubtitleList []Subtitle `xml:"subtitle"`
	}

	l := getLanguage(lang)

	searchUrl := "http://podnapisi.net/subtitles/search/old?sXML=1&sK=%s&sY=%s&sJ=%s"
	uri := fmt.Sprintf(searchUrl, url.QueryEscape(movie), year, l.ID)

	if category == CategoryTV || category == CategoryHDTV {
		if season != 0 {
			uri = uri + fmt.Sprintf("&sTS=%d&sTE=%d", season, episode)
		}
	}

	if verbose {
		log.Printf("SUB: GET %s\n", uri)
	}

	res, err := getResponse(uri, true)
	if err != nil {
		log.Printf("ERROR: getResponse %s\n", err.Error())
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("ERROR: ReadAll: %s\n", err.Error())
		return
	}

	var r Data
	err = xml.Unmarshal(body, &r)
	if err != nil {
		log.Printf("ERROR: Unmarshal: %s\n", err.Error())
		return
	}

	if len(r.SubtitleList) == 0 {
		return
	}

	for _, s := range r.SubtitleList {
		rel := strings.Fields(s.Release)
		var subtitleRelease string
		if len(rel) > 0 {
			subtitleRelease = rel[0]
		} else {
			continue
		}

		lid, _ := strconv.Atoi(l.ID)
		if s.LanguageID != lid {
			continue
		}

		score := getSubScore(torrentRelease, subtitleRelease)
		if score < scoreRequired {
			continue
		}

		if category == CategoryTV || category == CategoryHDTV {
			if season != s.TvSeason || episode != s.TvEpisode {
				continue
			}
		}

		downloadLink := fmt.Sprintf("http://podnapisi.net/subtitles/%s/download", s.Pid)

		s := TSubtitle{strconv.Itoa(s.Id), s.Title, s.Year, subtitleRelease, downloadLink, score}
		subtitles = append(subtitles, s)
	}
}

// opensubtitles.org
func opensubtitles(movie string, imdbId string, year string, torrentRelease string, lang string, category int, season int, episode int) {
	defer wgs.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("SUB: Recovered in opensubtitles")
		}
	}()

	o, err := osdb.NewClient()
	if err != nil {
		log.Printf("ERROR: NewClient: %s\n", err.Error())
	}

	err = o.LogIn(opensubsUser, opensubsPassword, "en")
	if err != nil {
		log.Printf("ERROR: LogIn: %s\n", err.Error())
	}

	o.UserAgent = opensubsUserAgent

	params := []interface{}{
		o.Token,
		[]map[string]string{},
	}

	l := getLanguage(lang)
	var listArgs []map[string]string
	listArgs = append(listArgs, map[string]string{"imdbid": imdbId})
	listArgs = append(listArgs, map[string]string{"sublanguageid": l.ISO_639_2})

	if category == CategoryTV || category == CategoryHDTV {
		if season != 0 {
			listArgs = append(listArgs, map[string]string{"season": strconv.Itoa(season)})
			listArgs = append(listArgs, map[string]string{"episode": strconv.Itoa(episode)})
		}
	}

	if verbose {
		log.Printf("SUB: XML-RPC http://opensubtitles.org %+v\n", listArgs)
	}

	params[1] = listArgs
	subs, err := o.SearchSubtitles(&params)
	if err != nil {
		log.Printf("ERROR: SearchSubtitles: %v\n", err.Error())
		return
	}

	if len(subs) == 0 {
		return
	}

	for _, sub := range subs {
		if sub.SubLanguageID != l.ISO_639_2 {
			continue
		}

		if !isSubValid(sub.SubFormat) {
			continue
		}

		subSumCD, _ := strconv.Atoi(sub.SubSumCD)
		if subSumCD > 1 {
			continue
		}

		score := getSubScore(torrentRelease, sub.MovieReleaseName)
		if score < scoreRequired {
			continue
		}

		if category == CategoryTV || category == CategoryHDTV {
			subSeason, _ := strconv.Atoi(sub.SeriesSeason)
			subEpisode, _ := strconv.Atoi(sub.SeriesEpisode)
			if season != subSeason || episode != subEpisode {
				continue
			}
		}

		s := TSubtitle{sub.IDSubtitleFile, sub.MovieName, sub.MovieYear, sub.MovieReleaseName, sub.ZipDownloadLink, score}
		subtitles = append(subtitles, s)
	}

}

// subscene.com subtitles
func subscene(movie string, year string, torrentRelease string, lang string, category int, season int, episode int) {
	defer wgs.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("SUB: Recovered in subscene")
		}
	}()

	searchUrl := "http://subscene.com/subtitles/release?q=%s"

	uri := fmt.Sprintf(searchUrl, url.QueryEscape(torrentRelease))

	if verbose {
		log.Printf("SUB: GET %s\n", uri)
	}

	doc, err := getDocument(uri, true)
	if err != nil {
		log.Printf("ERROR: getDocument %s", err.Error())
		return
	}

	tds := doc.Find(`td.a1`)

	if tds.Length() == 0 {
		return
	}

	subs := make([]TSubtitle, 0)

	var w sync.WaitGroup
	parseHTML := func(i int, td *goquery.Selection) {
		defer w.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Print("SUB: Recovered in parseHTML")
			}
		}()

		link := td.Find(`a`).First()
		href, _ := link.Attr("href")
		id := filepath.Base(href)

		subtitleLang := td.Find(`span`).First().Text()
		subtitleRelease := td.Find(`span`).Next().Text()
		subtitleYear := getYear(subtitleRelease)

		subtitleLang = strings.TrimSpace(subtitleLang)
		subtitleRelease = strings.TrimSpace(subtitleRelease)
		subtitleYear = strings.TrimSpace(subtitleYear)

		subtitleTitle := getTitle(subtitleRelease)

		if strings.ToLower(subtitleTitle) != strings.ToLower(movie) {
			return
		}

		if strings.ToLower(subtitleLang) != strings.ToLower(lang) {
			return
		}

		if subtitleYear != "" && subtitleYear != year {
			return
		}

		score := getSubScore(torrentRelease, subtitleRelease)
		if score < scoreRequired {
			return
		}

		if category == CategoryTV || category == CategoryHDTV {
			rs, _ := strconv.Atoi(getSeason(subtitleRelease))
			re, _ := strconv.Atoi(getEpisode(subtitleRelease))
			if season != rs || episode != re {
				return
			}
		}

		s := TSubtitle{id, subtitleTitle, subtitleYear, subtitleRelease, href, score}
		subs = append(subs, s)
	}

	w.Add(tds.Length())
	tds.Each(func(i int, s *goquery.Selection) {
		go parseHTML(i, s)
	})
	w.Wait()

	if len(subs) == 0 {
		return
	}

	sort.Sort(byScore(subs))
	s := subs[0]

	d, err := getDocument("http://subscene.com"+s.DownloadLink, true)
	if err != nil {
		log.Printf("ERROR: getDocument %s\n", err.Error())
		return
	}

	downloadHref := d.Find("#downloadButton")
	downloadLink, _ := downloadHref.Attr("href")

	sub := TSubtitle{"0", s.Title, s.Year, s.Release, "http://subscene.com" + downloadLink, s.Score}
	subtitles = append(subtitles, sub)
}

// Gets subtitle score
func getSubScore(torrentRelease string, subtitleRelease string) float64 {
	for _, char := range []string{".", "-", "_"} {
		torrentRelease = strings.Replace(torrentRelease, char, " ", -1)
		subtitleRelease = strings.Replace(subtitleRelease, char, " ", -1)
	}
	return smetrics.Jaro(torrentRelease, subtitleRelease)
}

// Checks if subtitle is valid
func isSubValid(format string) bool {
	formats := []string{"srt", "ass", "ssa"}
	if runtime.GOOS != "android" {
		formats = append(formats, "sub")
	}
	for _, f := range formats {
		if f == format {
			return true
		}
	}
	return false
}
