package bwtsearch

import (
	"bytes"
	"sort"
	"testing"
)

func TestPublicBuildSearchAndPersistence(t *testing.T) {
	idx := Build([]byte("abracadabra"))

	if got := idx.Count([]byte("abra")); got != 2 {
		t.Fatalf("Count(abra) = %d, want 2", got)
	}

	res, err := Search(idx, "abra|cad", 0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if res.TotalCount != 3 {
		t.Fatalf("TotalCount = %d, want 3", res.TotalCount)
	}

	positions := res.Positions(idx)
	sort.Ints(positions)
	want := []int{0, 4, 7}
	if len(positions) != len(want) {
		t.Fatalf("positions len = %d, want %d", len(positions), len(want))
	}
	for i := range want {
		if positions[i] != want[i] {
			t.Fatalf("positions[%d] = %d, want %d", i, positions[i], want[i])
		}
	}

	var buf bytes.Buffer
	if _, err := idx.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	loaded, err := ReadFrom(&buf)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}
	if got := loaded.Count([]byte("abra")); got != 2 {
		t.Fatalf("Count after reload = %d, want 2", got)
	}
}

func TestPublicCheckRejectsKleeneStar(t *testing.T) {
	if err := Check("ab*"); err == nil {
		t.Fatal("expected violation error for Kleene star")
	}
}

func TestNilIndexErrors(t *testing.T) {
	if _, err := Search(nil, "abra", 0); err == nil {
		t.Fatal("expected error for nil index")
	}

	var idx *Index
	var buf bytes.Buffer
	if _, err := idx.WriteTo(&buf); err == nil {
		t.Fatal("expected error writing nil index")
	}
}
