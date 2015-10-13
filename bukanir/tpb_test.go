package bukanir

import (
	"fmt"
	"testing"
)

func TestTop(t *testing.T) {
	pb := tpbInit(getHost())

	results, err := pb.Top(category_movies)
	if err != nil {
		t.Error(err)
	}

	if len(results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("%+v\n", results[0])
	}
}

func TestSearch(t *testing.T) {
	pb := tpbInit(getHost())

	results, err := pb.Search(t_name, 0)
	if err != nil {
		t.Error(err)
	}

	if len(results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("%+v\n", results[0])
	}
}
