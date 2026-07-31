package bwtsearch

import (
	"bytes"
	"sort"
	"testing"
)

// --- OccWaveletMatrix tests ------------------------------------------------

func TestBuildWithOptionsWaveletMatrix(t *testing.T) {
	idx := BuildWithOptions([]byte("abracadabra"), AlgorithmDoubling, OccWaveletMatrix)

	if got := idx.Count([]byte("abra")); got != 2 {
		t.Fatalf("Count(abra) = %d, want 2", got)
	}
	if got := idx.Count([]byte("xyz")); got != 0 {
		t.Fatalf("Count(xyz) = %d, want 0", got)
	}
}

func TestBuildWithOptionsWaveletMatrixMatchesBitvectors(t *testing.T) {
	text := []byte("the quick brown fox jumps over the lazy dog")
	idxBV := Build(text)
	idxWM := BuildWithOptions(text, AlgorithmSAIS, OccWaveletMatrix)

	patterns := []string{"the", "fox", "quick", "xyz", "dog"}
	for _, pat := range patterns {
		p := []byte(pat)
		if got, want := idxWM.Count(p), idxBV.Count(p); got != want {
			t.Errorf("WaveletMatrix Count(%q): got=%d bitvectors=%d", pat, got, want)
		}
		posWM := sortedInts(idxWM.Locate(p, 0))
		posBV := sortedInts(idxBV.Locate(p, 0))
		if !intSliceEq(posWM, posBV) {
			t.Errorf("WaveletMatrix Locate(%q): got=%v bitvectors=%v", pat, posWM, posBV)
		}
	}
}

func TestBuildWithOptionsWaveletMatrixPersistence(t *testing.T) {
	idx := BuildWithOptions([]byte("abracadabra"), AlgorithmDoubling, OccWaveletMatrix)

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
	if got := loaded.OccType(); got != OccWaveletMatrix {
		t.Fatalf("OccType after reload = %v, want OccWaveletMatrix", got)
	}
}

// --- OccRLBWT tests --------------------------------------------------------

func TestBuildWithOptionsRLBWT(t *testing.T) {
	idx := BuildWithOptions([]byte("abracadabra"), AlgorithmDoubling, OccRLBWT)

	if got := idx.Count([]byte("abra")); got != 2 {
		t.Fatalf("Count(abra) = %d, want 2", got)
	}
	if got := idx.Count([]byte("xyz")); got != 0 {
		t.Fatalf("Count(xyz) = %d, want 0", got)
	}
}

func TestBuildWithOptionsRLBWTMatchesBitvectors(t *testing.T) {
	texts := []string{
		"banana",
		"mississippi",
		"abracadabra",
		"the quick brown fox jumps over the lazy dog",
		"aaaaabbbbbccccc",
	}
	patterns := []string{"a", "an", "the", "ab", "ccc", "xyz"}

	for _, textStr := range texts {
		text := []byte(textStr)
		idxBV := Build(text)
		idxRL := BuildWithOptions(text, AlgorithmDoubling, OccRLBWT)

		for _, pat := range patterns {
			p := []byte(pat)
			if got, want := idxRL.Count(p), idxBV.Count(p); got != want {
				t.Errorf("text=%q RLBWT Count(%q): got=%d bitvectors=%d", textStr, pat, got, want)
			}
			posRL := sortedInts(idxRL.Locate(p, 0))
			posBV := sortedInts(idxBV.Locate(p, 0))
			if !intSliceEq(posRL, posBV) {
				t.Errorf("text=%q RLBWT Locate(%q): got=%v bitvectors=%v", textStr, pat, posRL, posBV)
			}
		}
	}
}

func TestBuildWithOptionsRLBWTPersistence(t *testing.T) {
	idx := BuildWithOptions([]byte("abracadabra"), AlgorithmSAIS, OccRLBWT)

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
	if got := loaded.OccType(); got != OccRLBWT {
		t.Fatalf("OccType after reload = %v, want OccRLBWT", got)
	}
}

func TestNumBWTRunsRepetitive(t *testing.T) {
	// A text of all 'a's: BWT is all 'a's followed by the sentinel, so 2 runs.
	idx := BuildWithOptions([]byte("aaaaaaaaaa"), AlgorithmDoubling, OccRLBWT)
	if got := idx.NumBWTRuns(); got != 2 {
		t.Errorf("NumBWTRuns for all-a = %d, want 2 (a-run + sentinel)", got)
	}
	// An alternating text should have many more runs than a repetitive one.
	idxAlt := Build([]byte("ababababab"))
	idxRep := Build([]byte("aaaaaaaaaa"))
	if idxAlt.NumBWTRuns() <= idxRep.NumBWTRuns() {
		t.Errorf("alternating (%d runs) should have more runs than repetitive (%d runs)",
			idxAlt.NumBWTRuns(), idxRep.NumBWTRuns())
	}
}

// --- BiIndex tests ---------------------------------------------------------

func TestBiIndexCountMatchesFMIndex(t *testing.T) {
	text := []byte("abracadabra")
	fmIdx := Build(text)
	biIdx := BuildBi(text)

	patterns := []string{"abra", "bra", "cadabra", "a", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		if got, want := biIdx.Count(p), fmIdx.Count(p); got != want {
			t.Errorf("BiIndex Count(%q) = %d, FM-index = %d", pat, got, want)
		}
	}
}

func TestBiIndexLocateMatchesFMIndex(t *testing.T) {
	text := []byte("mississippi")
	fmIdx := Build(text)
	biIdx := BuildBi(text)

	patterns := []string{"i", "is", "si", "iss", "p", "pp"}
	for _, pat := range patterns {
		p := []byte(pat)
		posBi := sortedInts(biIdx.Locate(p, 0))
		posFM := sortedInts(fmIdx.Locate(p, 0))
		if !intSliceEq(posBi, posFM) {
			t.Errorf("BiIndex Locate(%q): got=%v FM-index=%v", pat, posBi, posFM)
		}
	}
}

func TestBiIndexTextLen(t *testing.T) {
	text := []byte("hello world")
	bi := BuildBi(text)
	if got := bi.TextLen(); got != len(text) {
		t.Errorf("TextLen() = %d, want %d", got, len(text))
	}
}

func TestBiIndexContextAround(t *testing.T) {
	text := []byte("hello world")
	bi := BuildBi(text)
	ctx := bi.ContextAround(6, 5, 2)
	if len(ctx) == 0 {
		t.Error("ContextAround returned empty string")
	}
}

func TestBiIndexExtendLeft(t *testing.T) {
	// Verify that ExtendLeft on the full pattern gives the correct count.
	text := []byte("abracadabra")
	bi := BuildBi(text)
	ref := Build(text)

	pattern := []byte("abra")
	// Extend left character by character (backward search).
	biv := bi.FullInterval()
	for i := len(pattern) - 1; i >= 0; i-- {
		biv = bi.ExtendLeft(biv, pattern[i])
	}

	want := ref.Count(pattern)
	if got := biv.Size(); got != want {
		t.Errorf("ExtendLeft size = %d, want %d", got, want)
	}
}

func TestBiIndexExtendRight(t *testing.T) {
	// ExtendRight on a full pattern should give the same count as a forward search.
	text := []byte("abracadabra")
	bi := BuildBi(text)

	// Search "abra" by extending one character at a time to the right.
	biv := bi.FullInterval()
	for _, c := range []byte("abra") {
		biv = bi.ExtendRight(biv, c)
	}
	want := bi.Count([]byte("abra"))
	if got := biv.Size(); got != want {
		t.Errorf("ExtendRight size = %d, want %d (Count result)", got, want)
	}
}

func TestBiIndexPersistence(t *testing.T) {
	text := []byte("the quick brown fox jumps over the lazy dog")
	bi := BuildBiWithOptions(text, AlgorithmDoubling, OccBitvectors)

	var buf bytes.Buffer
	if _, err := bi.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	loaded, err := ReadBiFrom(&buf)
	if err != nil {
		t.Fatalf("ReadBiFrom failed: %v", err)
	}

	patterns := []string{"the", "fox", "dog", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		if got, want := loaded.Count(p), bi.Count(p); got != want {
			t.Errorf("after reload Count(%q): got=%d want=%d", pat, got, want)
		}
		posLoaded := sortedInts(loaded.Locate(p, 0))
		posOrig := sortedInts(bi.Locate(p, 0))
		if !intSliceEq(posLoaded, posOrig) {
			t.Errorf("after reload Locate(%q): got=%v want=%v", pat, posLoaded, posOrig)
		}
	}
}

func TestBiIndexWithWaveletMatrix(t *testing.T) {
	text := []byte("mississippi")
	bi := BuildBiWithOptions(text, AlgorithmSAIS, OccWaveletMatrix)
	ref := Build(text)

	patterns := []string{"i", "is", "p", "ippi"}
	for _, pat := range patterns {
		p := []byte(pat)
		if got, want := bi.Count(p), ref.Count(p); got != want {
			t.Errorf("BiIndex+WaveletMatrix Count(%q) = %d, want %d", pat, got, want)
		}
	}
}

func TestBiIndexFromFiles(t *testing.T) {
	texts := [][]byte{[]byte("hello"), []byte("world")}
	bi := BuildBiFromFiles(texts, nil)
	if got := bi.Count([]byte("hello")); got != 1 {
		t.Errorf("Count(hello) = %d, want 1", got)
	}
	if got := bi.Count([]byte("world")); got != 1 {
		t.Errorf("Count(world) = %d, want 1", got)
	}
}

// helpers

func sortedInts(s []int) []int {
	out := append([]int(nil), s...)
	sort.Ints(out)
	return out
}

func intSliceEq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
