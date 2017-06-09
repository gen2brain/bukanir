package bukanir

import (
	"strconv"
	"testing"
)

func TestSearchMovie(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.SearchMovie(tName)
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("no results")
	}
}

func TestSearchTv(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.SearchTv(teName)
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("no results")
	}
}

func TestMovieDetails(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.GetMovieDetails(strconv.Itoa(tId))
	if err != nil {
		t.Error(err)
	}

	if results.Id != tId {
		t.Error("id doesn't match")
	}
}

func TestTvDetails(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.GetTvDetails(strconv.Itoa(teId), -1)
	if err != nil {
		t.Error(err)
	}

	if results.Id != teId {
		t.Error("id doesn't match")
	}
}

func TestPopular(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.PopularMovies()
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("no results")
	}
}

func TestTopRated(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.TopRatedMovies()
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("no results")
	}
}

func TestGenres(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.Genres()
	if err != nil {
		t.Error(err)
	}

	if len(results.Genres) == 0 {
		t.Error("no results")
	}
}

func TestGenre(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.Genre(16, 1)
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("no results")
	}
}

func TestCast(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.MoviesWithCast(85, 1)
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("no results")
	}
}

func TestCrew(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.MoviesWithCrew(578, 1)
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("no results")
	}
}
