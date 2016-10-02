package bukanir

import (
	"fmt"
	"testing"
)

func TestTop(t *testing.T) {
	pb := NewTpb(getTpbHost())

	results, err := pb.Top(CategoryMovies)
	if err != nil {
		t.Error(err)
	}

	if len(results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("TPB Top:%+v\n", results[0])
	}
}

func TestSearch(t *testing.T) {
	pb := NewTpb(getTpbHost())

	results, err := pb.Search(t_name, 0, "201,207,205,208")
	if err != nil {
		t.Error(err)
	}

	if len(results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("TPB Search:%+v\n", results[0])
	}
}
