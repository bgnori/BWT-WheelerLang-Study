package bwtsearch

import (
	"bytes"
	"errors"
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

func TestPublicCheckRejectsUnsupportedAnchors(t *testing.T) {
	err := Check("^abra")
	if err == nil {
		t.Fatal("expected unsupported error for anchor")
	}
	var ue *UnsupportedError
	if !errors.As(err, &ue) {
		t.Fatalf("error type = %T, want *UnsupportedError", err)
	}
}

func TestPublicSearchRejectsUnsupportedAnchors(t *testing.T) {
	idx := Build([]byte("hello\nworld\nhello"))
	_, err := Search(idx, "^hello", 0)
	if err == nil {
		t.Fatal("expected unsupported error for anchor")
	}
	var ue *UnsupportedError
	if !errors.As(err, &ue) {
		t.Fatalf("error type = %T, want *UnsupportedError", err)
	}
}

func TestBuildFromFilesPanicsOnNullSeparator(t *testing.T) {
	texts := [][]byte{[]byte("hello"), []byte("world")}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for separator containing 0x00")
		}
	}()
	BuildFromFiles(texts, []byte{0x00})
}

func TestBuildFromFilesPanicsOnSeparatorWithEmbeddedNull(t *testing.T) {
	texts := [][]byte{[]byte("hello"), []byte("world")}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for separator containing 0x00")
		}
	}()
	BuildFromFiles(texts, []byte{'-', 0x00, '-'})
}

func TestNilIndexErrors(t *testing.T) {
	if _, err := Search(nil, "abra", 0); err == nil {
		t.Fatal("expected error for nil index")
	}

	if err := (*Index)(nil).Append([]byte("x")); err == nil {
		t.Fatal("expected error appending to nil index")
	}

	var idx *Index
	var buf bytes.Buffer
	if _, err := idx.WriteTo(&buf); err == nil {
		t.Fatal("expected error writing nil index")
	}
}

func TestNilIndexZeroValues(t *testing.T) {
	var idx *Index

	if got := idx.Count([]byte("abc")); got != 0 {
		t.Fatalf("nil Count = %d, want 0", got)
	}
	if got := idx.Locate([]byte("abc"), 0); got != nil {
		t.Fatalf("nil Locate = %v, want nil", got)
	}
	if got := idx.TextLen(); got != 0 {
		t.Fatalf("nil TextLen = %d, want 0", got)
	}
	if got := idx.SALen(); got != 0 {
		t.Fatalf("nil SALen = %d, want 0", got)
	}
	if got := idx.SAAt(0); got != 0 {
		t.Fatalf("nil SAAt = %d, want 0", got)
	}
	if got := idx.AlphabetSize(); got != 0 {
		t.Fatalf("nil AlphabetSize = %d, want 0", got)
	}
	if got := idx.NumBWTRuns(); got != 0 {
		t.Fatalf("nil NumBWTRuns = %d, want 0", got)
	}
	if got := idx.BWT(); got != nil {
		t.Fatalf("nil BWT = %v, want nil", got)
	}
	if got := idx.ContextAround(0, 0, 0); got != "" {
		t.Fatalf("nil ContextAround = %q, want empty", got)
	}
	if got := idx.WheelerGraphMermaid(0); got != "" {
		t.Fatalf("nil WheelerGraphMermaid = %q, want empty", got)
	}
	if got := idx.OccType(); got != OccBitvectors {
		t.Fatalf("nil OccType = %v, want OccBitvectors", got)
	}
}

func TestBuildNilAndEmpty(t *testing.T) {
	for _, tc := range []struct {
		name string
		text []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			idx := Build(tc.text)
			if idx == nil {
				t.Fatal("Build returned nil index")
			}
			if got := idx.TextLen(); got != 0 {
				t.Fatalf("TextLen = %d, want 0", got)
			}
			if got := idx.SALen(); got != 1 {
				t.Fatalf("SALen = %d, want 1 (sentinel only)", got)
			}
			if got := idx.Count([]byte("a")); got != 0 {
				t.Fatalf("Count(a) = %d, want 0", got)
			}
			res, err := Search(idx, "a", 0)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}
			if res.TotalCount != 0 {
				t.Fatalf("Search TotalCount = %d, want 0", res.TotalCount)
			}
		})
	}
}

func TestBuildWithOptionsWavelet(t *testing.T) {
	idx := BuildWithOptions([]byte("abracadabra"), AlgorithmDoubling, OccWaveletTree)

	if got := idx.Count([]byte("abra")); got != 2 {
		t.Fatalf("Count(abra) = %d, want 2", got)
	}
	if got := idx.Count([]byte("xyz")); got != 0 {
		t.Fatalf("Count(xyz) = %d, want 0", got)
	}
}

func TestBuildWithOptionsWaveletMatchesBitvectors(t *testing.T) {
	text := []byte("the quick brown fox jumps over the lazy dog")
	idxBV := Build(text)
	idxWT := BuildWithOptions(text, AlgorithmSAIS, OccWaveletTree)

	patterns := []string{"the", "fox", "quick", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		if got, want := idxWT.Count(p), idxBV.Count(p); got != want {
			t.Errorf("Count(%q): wavelet=%d bitvectors=%d", pat, got, want)
		}
	}
}

func TestBuildWithOptionsWaveletPersistence(t *testing.T) {
	idx := BuildWithOptions([]byte("abracadabra"), AlgorithmDoubling, OccWaveletTree)

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

func TestBuildWithOptionsEliasFano(t *testing.T) {
	idx := BuildWithOptions([]byte("abracadabra"), AlgorithmDoubling, OccEliasFano)

	if got := idx.Count([]byte("abra")); got != 2 {
		t.Fatalf("Count(abra) = %d, want 2", got)
	}
	if got := idx.Count([]byte("xyz")); got != 0 {
		t.Fatalf("Count(xyz) = %d, want 0", got)
	}
}

func TestBuildWithOptionsEliasFanoPersistence(t *testing.T) {
	idx := BuildWithOptions([]byte("abracadabra"), AlgorithmSAIS, OccEliasFano)

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

func TestPublicAppend(t *testing.T) {
	idx := Build([]byte("hello"))
	if err := idx.Append([]byte(" world")); err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if got := idx.Count([]byte("hello world")); got != 1 {
		t.Fatalf("Count(hello world) = %d, want 1", got)
	}
	if got := idx.Count([]byte("world")); got != 1 {
		t.Fatalf("Count(world) = %d, want 1", got)
	}
}

func TestPublicAppendWavelet(t *testing.T) {
	idx := BuildWithOptions([]byte("abc"), AlgorithmSAIS, OccWaveletTree)
	if err := idx.Append([]byte("defabc")); err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if got := idx.Count([]byte("abc")); got != 2 {
		t.Fatalf("Count(abc) = %d, want 2", got)
	}
}
