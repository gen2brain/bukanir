package bukanir

import (
	"fmt"
	"testing"
)

func TestSearchEztv(t *testing.T) {
	ez := NewEztv("eztv.ag")

	results, err := ez.Search(te_name)
	if err != nil {
		t.Error(err)
	}

	if len(results) == 0 {
		t.Error("FAIL")
	} else {
		fmt.Printf("EZTV Search:%+v\n", results[0])
	}
}
