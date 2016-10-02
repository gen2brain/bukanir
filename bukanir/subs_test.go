package bukanir

import (
	"fmt"
	"testing"
)

var (
	t_id              = 348
	t_name     string = "Alien"
	t_year     string = "1979"
	t_release  string = "Alien.Directors.Cut.1979.1080p.BRrip.x264.GAZ.YIFY"
	t_language string = "english"
	t_category int    = 201
	t_imdbid   string = "0078748"

	te_id              = 1399
	te_name     string = "Game of Thrones"
	te_year     string = "2011"
	te_release  string = "Game.of.Thrones.S01E03.Lord.Snow.HDTV.XviD-FQM"
	te_language string = "english"
	te_category int    = 205
	te_season   int    = 1
	te_episode  int    = 3
	te_imdbid   string = "0944947"
)

func TestPodnapisi(t *testing.T) {
	wgs.Add(1)
	go podnapisi(t_name, t_year, t_release, t_language, t_category, 0, 0)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("Podnapisi:%+v\n", subtitles[0])
		subtitles = make([]TSubtitle, 0)
	}

	wgs.Add(1)
	go podnapisi(te_name, te_year, te_release, te_language, te_category, te_season, te_episode)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("Podnapisi:%+v\n", subtitles[0])
		subtitles = make([]TSubtitle, 0)
	}
}

func TestOpensubtitles(t *testing.T) {
	wgs.Add(1)
	go opensubtitles(t_name, t_imdbid, t_year, t_release, t_language, t_category, 0, 0)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("OpenSubtitles:%+v\n", subtitles[0])
		subtitles = make([]TSubtitle, 0)
	}

	wgs.Add(1)
	go opensubtitles(te_name, te_imdbid, te_year, te_release, te_language, te_category, te_season, te_episode)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("OpenSubtitles:%+v\n", subtitles[0])
		subtitles = make([]TSubtitle, 0)
	}
}

func TestSubscene(t *testing.T) {
	wgs.Add(1)
	go subscene(t_name, t_year, t_release, t_language, t_category, 0, 0)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("SubScene:%+v\n", subtitles[0])
		subtitles = make([]TSubtitle, 0)
	}

	wgs.Add(1)
	go subscene(te_name, te_year, te_release, te_language, te_category, te_season, te_episode)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("SubScene:%+v\n", subtitles[0])
		subtitles = make([]TSubtitle, 0)
	}

	fmt.Println()
}
