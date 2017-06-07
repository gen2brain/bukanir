package bukanir

import (
	"testing"
)

func TestTopTpb(t *testing.T) {
	pb := NewTpb(getTpbHost())

	results, err := pb.Top(CategoryMovies)
	if err != nil {
		t.Error(err)
	}

	if len(results) == 0 {
		t.Error("no results")
	}
}

func TestSearchTpb(t *testing.T) {
	pb := NewTpb(getTpbHost())

	results, err := pb.Search(tName, 0, "201,207,205,208")
	if err != nil {
		t.Error(err)
	}

	if len(results) == 0 {
		t.Error("no results")
	}
}
