package bukanir

import (
	"log"
	"net/url"
	"strings"
	"sync/atomic"
)

var verbose atomic.Bool

// SetVerbose toggles operational logging via the standard logger.
func SetVerbose(v bool) { verbose.Store(v) }

func logf(format string, args ...any) {
	if verbose.Load() {
		log.Printf(format, args...)
	}
}

// sourceTag derives a short log prefix from a request URL's host.
func sourceTag(rawurl string) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		return "HTTP"
	}
	h := strings.ToLower(u.Hostname())
	switch {
	case strings.Contains(h, "themoviedb"):
		return "TMDB"
	case strings.Contains(h, "eztv"):
		return "EZTV"
	case strings.Contains(h, "apibay"), strings.Contains(h, "piratebay"):
		return "TPB"
	case strings.Contains(h, "opensubtitles"), strings.Contains(h, "subdl"),
		strings.Contains(h, "gestdown"), strings.Contains(h, "moviesubtitles"),
		strings.Contains(h, "yifysubtitles"):
		return "SUB"
	}
	return "HTTP"
}
