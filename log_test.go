package bukanir

import "testing"

func TestSourceTag(t *testing.T) {
	cases := map[string]string{
		"https://api.themoviedb.org/3/movie/1?api_key=x": "TMDB",
		"https://eztvx.to/api/get-torrents":              "EZTV",
		"https://apibay.org/q.php?q=x":                   "TPB",
		"https://api.opensubtitles.com/api/v1/login":     "SUB",
		"https://dl.subdl.com/subtitle/x.zip":            "SUB",
		"https://example.com/x":                          "HTTP",
	}
	for url, want := range cases {
		if got := sourceTag(url); got != want {
			t.Errorf("sourceTag(%q) = %q, want %q", url, got, want)
		}
	}
}
