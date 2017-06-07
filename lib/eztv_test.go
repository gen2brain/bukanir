package bukanir

import (
	"testing"
)

func TestSearchEztv(t *testing.T) {
	ez := NewEztv(getEztvHost())

	results, err := ez.Search(teName)
	if err != nil {
		t.Error(err)
	}

	if len(results) == 0 {
		t.Error("no results")
	}
}
