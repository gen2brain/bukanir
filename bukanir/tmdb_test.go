package bukanir

import (
	"fmt"
	"strconv"
	"testing"
)

func TestSearchMovie(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.SearchMovie(t_name)
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("Movie:%+v\n", results.Results[0])
	}
}

func TestSearchTv(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.SearchTv(te_name)
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("TV:%+v\n", results.Results[0])
	}
}

func TestMovieDetails(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.GetMovieDetails(strconv.Itoa(t_id))
	if err != nil {
		t.Error(err)
	}

	if results.Id != t_id {
		t.Error("FAIL")
	} else {
		fmt.Printf("\nMovie Overview:%+v\n\n", results)
	}
}

func TestTvDetails(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.GetTvDetails(strconv.Itoa(te_id), -1)
	if err != nil {
		t.Error(err)
	}

	if results.Id != te_id {
		t.Error("FAIL")
	} else {
		fmt.Printf("\nTV Overview:%+v\n\n", results)
	}
}

func TestPopular(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.PopularMovies()
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("Popular:%+v\n", results.Results[0])
	}
}

func TestTopRated(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.TopRatedMovies()
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("Top Rated:%+v\n", results.Results[0])
	}
}

func TestGenres(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.Genres()
	if err != nil {
		t.Error(err)
	}

	if len(results.Genres) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("Genres:%+v\n", results.Genres)
	}
}

func TestGenre(t *testing.T) {
	md := NewTmdb(tmdbApiKey)

	results, err := md.Genre(16, 1)
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("Genre:%+v\n", results.Results[0])
	}
}
