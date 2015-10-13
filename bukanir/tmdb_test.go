package bukanir

import (
	"fmt"
	"strconv"
	"testing"
)

func TestSearchMovie(t *testing.T) {
	md := tmdbInit(tmdbApiKey)

	results, err := md.SearchMovie(t_name)
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("%+v\n", results.Results[0])
	}
}

func TestSearchTv(t *testing.T) {
	md := tmdbInit(tmdbApiKey)

	results, err := md.SearchTv(te_name)
	if err != nil {
		t.Error(err)
	}

	if len(results.Results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("%+v\n", results.Results[0])
	}
}

func TestMovieDetails(t *testing.T) {
	md := tmdbInit(tmdbApiKey)

	results, err := md.GetMovieDetails(strconv.Itoa(t_id))
	if err != nil {
		t.Error(err)
	}

	if results.Id != t_id {
		t.Error("FAIL")
	} else {
		fmt.Printf("Overview:%s\n", results.Overview)
	}
}

func TestTvDetails(t *testing.T) {
	md := tmdbInit(tmdbApiKey)

	results, err := md.GetTvDetails(strconv.Itoa(te_id), -1)
	if err != nil {
		t.Error(err)
	}

	if results.Id != te_id {
		t.Error("FAIL")
	} else {
		fmt.Printf("Overview:%s\n", results.Overview)
	}
}
