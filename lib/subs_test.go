package bukanir

import (
	"testing"
)

var (
	tId              = 348
	tName     string = "Alien"
	tYear     string = "1979"
	tRelease  string = "Alien.Directors.Cut.1979.1080p.BRrip.x264.GAZ.YIFY"
	tLanguage string = "english"
	tCategory int    = 201
	tImdbId   string = "0078748"

	teId              = 1399
	teName     string = "Game of Thrones"
	teYear     string = "2011"
	teRelease  string = "Game.of.Thrones.S01E03.Lord.Snow.HDTV.XviD-FQM"
	teLanguage string = "english"
	teCategory int    = 205
	te_season  int    = 1
	te_episode int    = 3
	teImdbId   string = "0944947"
)

func TestPodnapisi(t *testing.T) {
	wgs.Add(1)
	go podnapisi(tName, tYear, tRelease, tLanguage, tCategory, 0, 0)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("no results")
	}

	wgs.Add(1)
	go podnapisi(teName, teYear, teRelease, teLanguage, teCategory, te_season, te_episode)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("no results")
	}
}

func TestOpensubtitles(t *testing.T) {
	wgs.Add(1)
	go opensubtitles(tName, tImdbId, tYear, tRelease, tLanguage, tCategory, 0, 0)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("no results")
	}

	wgs.Add(1)
	go opensubtitles(teName, teImdbId, teYear, teRelease, teLanguage, teCategory, te_season, te_episode)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("no results")
	}
}

func TestSubscene(t *testing.T) {
	wgs.Add(1)
	go subscene(tName, tYear, tRelease, tLanguage, tCategory, 0, 0)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("no results")
	}

	wgs.Add(1)
	go subscene(teName, teYear, teRelease, teLanguage, teCategory, te_season, te_episode)
	wgs.Wait()

	if len(subtitles) == 0 {
		t.Error("no results")
	}
}
