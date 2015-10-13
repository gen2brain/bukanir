package bukanir

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/oz/osdb"
	"github.com/xrash/smetrics"
)

type language struct {
	Name      string
	ISO_639_2 string
	ID        string
}

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

func podnapisi(movie string, year string, torrentRelease string, lang string, category int, season int, episode int) {
	defer wgs.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in podnapisi: ", r)
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

	searchUrl := podnapisiUrl + "search/old?sXML=1&sK=%s&sY=%s&sJ=%s"
	uri := fmt.Sprintf(searchUrl, url.QueryEscape(movie), year, l.ID)

	if category == category_tv || category == category_hdtv {
		if season != 0 {
			uri = uri + fmt.Sprintf("&sTS=%d&sTE=%d", season, episode)
		}
	}

	if verbose {
		log.Printf("Get %s\n", uri)
	}

	res, err := httpGetResponse(uri)
	if err != nil {
		log.Printf("Error httpGetResponse %s: %v\n", uri, err.Error())
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading body: %v\n", err.Error())
		return
	}

	var r Data
	err = xml.Unmarshal(body, &r)
	if err != nil {
		log.Printf("Error unmarshalling response: %v\n", err.Error())
		return
	}

	if len(r.SubtitleList) == 0 {
		return
	}

	scoreRequired := 0.7
	if lang != "english" {
		scoreRequired = 0.75
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

		score := getScore(torrentRelease, subtitleRelease)
		if score < scoreRequired {
			continue
		}

		if category == category_tv || category == category_hdtv {
			if season != s.TvSeason || episode != s.TvEpisode {
				continue
			}
		}

		downloadLink := fmt.Sprintf(podnapisiUrl+"%s/download", s.Pid)

		s := subtitle{strconv.Itoa(s.Id), s.Title, s.Year, subtitleRelease, downloadLink, score}
		subtitles = append(subtitles, s)
	}
}

func opensubtitles(movie string, imdbId string, year string, torrentRelease string, lang string, category int, season int, episode int) {
	defer wgs.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in opensubtitles: ", r)
		}
	}()

	o, err := osdb.NewClient()
	if err != nil {
		log.Printf("Error NewClient: %v\n", err.Error())
	}

	err = o.LogIn(opensubsUser, opensubsPassword, "en")
	if err != nil {
		log.Printf("Error LogIn: %v\n", err.Error())
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

	if category == category_tv || category == category_hdtv {
		if season != 0 {
			listArgs = append(listArgs, map[string]string{"season": strconv.Itoa(season)})
			listArgs = append(listArgs, map[string]string{"episode": strconv.Itoa(episode)})
		}
	}

	if verbose {
		log.Printf("XmlRpc %+v\n", listArgs)
	}

	params[1] = listArgs
	subs, err := o.SearchSubtitles(&params)
	if err != nil {
		log.Printf("Error SearchSubtitles: %v\n", err.Error())
		return
	}

	if len(subs) == 0 {
		return
	}

	scoreRequired := 0.7
	if lang != "english" {
		scoreRequired = 0.75
	}

	for _, sub := range subs {
		if sub.SubLanguageID != l.ISO_639_2 {
			continue
		}

		if !isValidFormat(sub.SubFormat) {
			continue
		}

		subSumCD, _ := strconv.Atoi(sub.SubSumCD)
		if subSumCD > 1 {
			continue
		}

		score := getScore(torrentRelease, sub.MovieReleaseName)
		if score < scoreRequired {
			continue
		}

		if category == category_tv || category == category_hdtv {
			subSeason, _ := strconv.Atoi(sub.SeriesSeason)
			subEpisode, _ := strconv.Atoi(sub.SeriesEpisode)
			if season != subSeason || episode != subEpisode {
				continue
			}
		}

		s := subtitle{sub.IDSubtitleFile, sub.MovieName, sub.MovieYear, sub.MovieReleaseName, sub.ZipDownloadLink, score}
		subtitles = append(subtitles, s)
	}

}

func subscene(movie string, year string, torrentRelease string, lang string, category int, season int, episode int) {
	defer wgs.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("Recovered in subscene: ", r)
		}
	}()

	searchUrl := subsceneUrl + "/subtitles/release?q=%s"

	uri := fmt.Sprintf(searchUrl, url.QueryEscape(torrentRelease))

	doc, err := getDocument(uri)
	if err != nil {
		log.Printf("Error getDocument %s: %v", uri, err.Error())
		return
	}

	tds := doc.Find(`td.a1`)

	if tds.Length() == 0 {
		return
	}

	scoreRequired := 0.7
	if lang != "english" {
		scoreRequired = 0.75
	}

	subs := make([]subtitle, 0)

	var w sync.WaitGroup
	parseHTML := func(i int, td *goquery.Selection) {
		defer w.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Print("Recovered in parseHTML: ", r)
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

		score := getScore(torrentRelease, subtitleRelease)
		if score < scoreRequired {
			return
		}

		if category == category_tv || category == category_hdtv {
			rs, _ := strconv.Atoi(getSeason(subtitleRelease))
			re, _ := strconv.Atoi(getEpisode(subtitleRelease))
			if season != rs || episode != re {
				return
			}
		}

		s := subtitle{id, subtitleTitle, subtitleYear, subtitleRelease, href, score}
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

	d, err := getDocument(subsceneUrl + s.DownloadLink)
	if err != nil {
		log.Printf("Error getDocument %s: %v", subsceneUrl+s.DownloadLink, err.Error())
		return
	}

	downloadHref := d.Find("#downloadButton")
	downloadLink, _ := downloadHref.Attr("href")

	sub := subtitle{"0", s.Title, s.Year, s.Release, subsceneUrl + downloadLink, s.Score}
	subtitles = append(subtitles, sub)
}

func getScore(torrentRelease string, subtitleRelease string) float64 {
	for _, char := range []string{".", "-", "_"} {
		torrentRelease = strings.Replace(torrentRelease, char, " ", -1)
		subtitleRelease = strings.Replace(subtitleRelease, char, " ", -1)
	}
	return smetrics.Jaro(torrentRelease, subtitleRelease)
}

func isValidFormat(format string) bool {
	formats := []string{"srt", "ass", "ssa", "sub"}
	for _, f := range formats {
		if f == format {
			return true
		}
	}
	return false
}
