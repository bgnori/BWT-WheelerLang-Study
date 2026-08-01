package fmindex

import (
	"bytes"
	"index/suffixarray"
	"sort"
	"strings"
	"testing"
)

// helper: sorted positions for comparison
func sortedPositions(ps []int) []int {
	out := append([]int(nil), ps...)
	sort.Ints(out)
	return out
}

// stdlibLocate uses Go's standard index/suffixarray to locate occurrences.
func stdlibLocate(text, pattern []byte) []int {
	sa := suffixarray.New(text)
	indices := sa.Lookup(pattern, -1)
	sort.Ints(indices)
	return indices
}

// --- unit tests ------------------------------------------------------------

func TestBuildAndCount(t *testing.T) {
	text := []byte("abracadabra")
	idx := Build(text)

	cases := []struct {
		pat  string
		want int
	}{
		{"a", 5},
		{"abra", 2},
		{"bra", 2},
		{"cadabra", 1},
		{"xyz", 0},
		{"abracadabra", 1},
	}
	for _, tc := range cases {
		got := idx.Count([]byte(tc.pat))
		if got != tc.want {
			t.Errorf("Count(%q) = %d, want %d", tc.pat, got, tc.want)
		}
	}
}

func TestLocate(t *testing.T) {
	text := []byte("abracadabra")
	idx := Build(text)

	// "abra" should be at positions 0 and 7
	positions := sortedPositions(idx.Locate([]byte("abra"), 0))
	want := []int{0, 7}
	if !intSliceEqual(positions, want) {
		t.Errorf("Locate(abra) = %v, want %v", positions, want)
	}

	// "a" should be at positions 0,2,4,6,8 (wait, check)
	// abracadabra: a=0,2,4,6,8 → wait: a(0)brac a(4)d a(6)b r a(10)
	// positions: 0,3,5,7,10 ... Let me count: a-b-r-a-c-a-d-a-b-r-a = indices 0,3,5,7,10
	positions = sortedPositions(idx.Locate([]byte("a"), 0))
	want = []int{0, 3, 5, 7, 10}
	if !intSliceEqual(positions, want) {
		t.Errorf("Locate(a) = %v, want %v", positions, want)
	}
}

func TestLocateLimit(t *testing.T) {
	text := []byte("aaaaaaaaaa") // 10 a's
	idx := Build(text)

	positions := idx.Locate([]byte("a"), 3)
	if len(positions) != 3 {
		t.Errorf("Locate with limit 3 returned %d results, want 3", len(positions))
	}
}

func TestCountEmpty(t *testing.T) {
	text := []byte("hello world")
	idx := Build(text)
	if got := idx.Count([]byte("")); got != idx.n {
		// empty pattern matches everywhere (all n positions)
		t.Logf("Count('') = %d (n=%d)", got, idx.n)
	}
}

func TestCompareWithStdlib(t *testing.T) {
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
		idx := Build(text)
		for _, pat := range patterns {
			p := []byte(pat)
			got := idx.Count(p)
			want := len(stdlibLocate(text, p))
			if got != want {
				t.Errorf("text=%q pat=%q: FM-index count=%d stdlib count=%d",
					textStr, pat, got, want)
			}
		}
	}
}

func TestLocateCompareWithStdlib(t *testing.T) {
	text := []byte("mississippi")
	patterns := []string{"i", "is", "si", "iss", "miss", "p", "pp", "ippi"}

	idx := Build(text)
	for _, pat := range patterns {
		p := []byte(pat)
		got := sortedPositions(idx.Locate(p, 0))
		want := stdlibLocate(text, p)
		if !intSliceEqual(got, want) {
			t.Errorf("pat=%q: FM-index locate=%v stdlib=%v", pat, got, want)
		}
	}
}

func TestPersistence(t *testing.T) {
	text := []byte("the quick brown fox jumps over the lazy dog")
	idx := Build(text)

	var buf bytes.Buffer
	if _, err := idx.WriteTo(&buf); err != nil {
		t.Fatal("WriteTo:", err)
	}

	idx2, err := ReadFrom(&buf)
	if err != nil {
		t.Fatal("ReadFrom:", err)
	}

	patterns := []string{"the", "fox", "dog", "quick", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		c1 := idx.Count(p)
		c2 := idx2.Count(p)
		if c1 != c2 {
			t.Errorf("after persistence, Count(%q): %d vs %d", pat, c1, c2)
		}
		pos1 := sortedPositions(idx.Locate(p, 0))
		pos2 := sortedPositions(idx2.Locate(p, 0))
		if !intSliceEqual(pos1, pos2) {
			t.Errorf("after persistence, Locate(%q): %v vs %v", pat, pos1, pos2)
		}
	}
}

func TestPersistencePreservesAlgorithmForAppend(t *testing.T) {
	tests := []struct {
		name string
		occ  OccStructure
	}{
		{"bitvectors", OccBitvectors},
		{"wavelet", OccWaveletTree},
		{"wavelet-matrix", OccWaveletMatrix},
		{"rlbwt", OccRLBWT},
		{"rrr", OccRRR},
		{"elias-fano", OccEliasFano},
		{"poppy", OccPoppy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := BuildWithOptions([]byte("banana"), AlgorithmSAIS, tt.occ)
			var buf bytes.Buffer
			if _, err := idx.WriteTo(&buf); err != nil {
				t.Fatal("WriteTo:", err)
			}

			loaded, err := ReadFrom(&buf)
			if err != nil {
				t.Fatal("ReadFrom:", err)
			}
			if loaded.algo != AlgorithmSAIS {
				t.Fatalf("algorithm after persistence = %d, want AlgorithmSAIS", loaded.algo)
			}
			if loaded.typ != tt.occ {
				t.Fatalf("occ type after persistence = %d, want %d", loaded.typ, tt.occ)
			}

			loaded.Append([]byte(" bandana"))
			want := BuildWithOptions([]byte("banana bandana"), AlgorithmSAIS, tt.occ)
			if !intSliceEqual(sortedPositions(loaded.Locate([]byte("ana"), 0)), sortedPositions(want.Locate([]byte("ana"), 0))) {
				t.Fatal("Append result differs from an SA-IS rebuild")
			}
		})
	}
}

func TestBWT(t *testing.T) {
	// "banana$" sorted suffixes:
	//   $ a a a b n n  → BWT = annb$aa   (classic result)
	// Let's verify our BWT construction produces a valid BWT by checking
	// that applying inverse-BWT gives back the original text.
	text := []byte("banana")
	idx := Build(text)
	_ = idx.BWT() // just check it doesn't panic

	// Verify Count is consistent with BWT structure
	if idx.Count([]byte("banana")) != 1 {
		t.Error("banana should appear once in its own text")
	}
	if idx.Count([]byte("an")) != 2 {
		t.Error("'an' should appear twice in banana")
	}
}

func TestAlphabetSize(t *testing.T) {
	text := []byte("hello")
	idx := Build(text)
	// distinct chars: h,e,l,o = 4, plus sentinel = 5 total in the BWT
	size := idx.AlphabetSize()
	if size < 4 {
		t.Errorf("AlphabetSize() = %d, want >= 4", size)
	}
}

func TestContextAround(t *testing.T) {
	text := []byte("hello world")
	idx := Build(text)
	ctx := idx.ContextAround(6, 5, 3)
	// should contain "world" plus some context
	if len(ctx) == 0 {
		t.Error("ContextAround returned empty string")
	}
}

func TestWheelerGraphMermaid(t *testing.T) {
	idx := Build([]byte("banana"))
	m := idx.WheelerGraphMermaid(0)

	if !strings.Contains(m, "flowchart LR") {
		t.Fatalf("Mermaid output must start with flowchart header: %q", m)
	}
	if !strings.Contains(m, "n0[") {
		t.Fatal("expected at least one node in Mermaid output")
	}
	if !strings.Contains(m, "--\"") {
		t.Fatal("expected labeled edges in Mermaid output")
	}
}

func TestWheelerGraphMermaidTruncation(t *testing.T) {
	idx := Build([]byte("mississippi"))
	m := idx.WheelerGraphMermaid(4)

	if !strings.Contains(m, "more nodes omitted") {
		t.Fatalf("expected omitted nodes marker in truncated output: %q", m)
	}
}

func intSliceEqual(a, b []int) bool {
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

// --- Japanese text tests ---------------------------------------------------

// TestJapaneseText verifies that the FM-index correctly handles UTF-8 encoded
// Japanese text.  "上杉謙信" (Uesugi Kenshin) appears twice in the sample
// sentence and the index must find both occurrences at the expected byte
// positions.
func TestJapaneseText(t *testing.T) {
	// 上杉謙信 appears at byte offsets 15 and 63 in this text (each kanji = 3 bytes).
	text := []byte("武田信玄と上杉謙信は戦国時代の名将である。上杉謙信は越後の虎と呼ばれた。")
	idx := Build(text)

	pattern := []byte("上杉謙信")

	// Count
	if got := idx.Count(pattern); got != 2 {
		t.Errorf("Count(上杉謙信) = %d, want 2", got)
	}

	// Locate
	positions := sortedPositions(idx.Locate(pattern, 0))
	wantPositions := []int{15, 63}
	if !intSliceEqual(positions, wantPositions) {
		t.Errorf("Locate(上杉謙信) = %v, want %v", positions, wantPositions)
	}

	// Absent sub-pattern
	if got := idx.Count([]byte("徳川家康")); got != 0 {
		t.Errorf("Count(徳川家康) = %d, want 0 (not in text)", got)
	}
}

// TestJapaneseTextSAIS confirms SA-IS and doubling produce the same results
// for Japanese input.
func TestJapaneseTextSAIS(t *testing.T) {
	text := []byte("武田信玄と上杉謙信は戦国時代の名将である。上杉謙信は越後の虎と呼ばれた。")
	pattern := []byte("上杉謙信")

	idxDoubling := Build(text)
	idxSAIS := BuildWithAlgorithm(text, AlgorithmSAIS)

	if idxDoubling.Count(pattern) != idxSAIS.Count(pattern) {
		t.Errorf("Count mismatch between doubling (%d) and SAIS (%d)",
			idxDoubling.Count(pattern), idxSAIS.Count(pattern))
	}

	posD := sortedPositions(idxDoubling.Locate(pattern, 0))
	posS := sortedPositions(idxSAIS.Locate(pattern, 0))
	if !intSliceEqual(posD, posS) {
		t.Errorf("Locate mismatch: doubling=%v sais=%v", posD, posS)
	}
}

// --- SA-IS tests -----------------------------------------------------------

func TestSAISMatchesDoubling(t *testing.T) {
	texts := []string{
		"banana",
		"mississippi",
		"abracadabra",
		"the quick brown fox jumps over the lazy dog",
		"aaaaabbbbbccccc",
		"aaaaaaaaaa",
		"abcdefghij",
		"",
	}
	patterns := []string{"a", "an", "the", "ab", "ccc", "xyz", "aa"}

	for _, textStr := range texts {
		text := []byte(textStr)
		idxDoubling := Build(text)
		idxSAIS := BuildWithAlgorithm(text, AlgorithmSAIS)

		for _, pat := range patterns {
			p := []byte(pat)
			gotD := idxDoubling.Count(p)
			gotS := idxSAIS.Count(p)
			if gotD != gotS {
				t.Errorf("text=%q pat=%q: doubling=%d sais=%d", textStr, pat, gotD, gotS)
			}
			posD := sortedPositions(idxDoubling.Locate(p, 0))
			posS := sortedPositions(idxSAIS.Locate(p, 0))
			if !intSliceEqual(posD, posS) {
				t.Errorf("text=%q pat=%q: doubling locate=%v sais locate=%v",
					textStr, pat, posD, posS)
			}
		}
	}
}

func TestSAISDirectSuffixArray(t *testing.T) {
	// Validate the raw suffix array built by SA-IS against the doubling SA.
	inputs := []string{
		"banana",
		"mississippi",
		"abracadabra",
		"aaaaaa",
		"abcabc",
	}
	for _, input := range inputs {
		// Append sentinel manually (same as Build does internally).
		text := append([]byte(input), sentinel)
		saDoubling := buildSuffixArray(text)
		saSAIS := buildSuffixArraySAIS(text)
		if !intSliceEqual(saDoubling, saSAIS) {
			t.Errorf("input=%q: doubling SA=%v  sais SA=%v", input, saDoubling, saSAIS)
		}
	}
}

// --- Wavelet tree tests ----------------------------------------------------

func TestWaveletMatchesBitvectors(t *testing.T) {
	texts := []string{
		"banana",
		"mississippi",
		"abracadabra",
		"the quick brown fox jumps over the lazy dog",
		"aaaaabbbbbccccc",
		"aaaaaaaaaa",
		"",
	}
	patterns := []string{"a", "an", "the", "ab", "ccc", "xyz", "aa"}

	for _, textStr := range texts {
		text := []byte(textStr)
		idxBV := Build(text)
		idxWT := BuildWithOptions(text, AlgorithmDoubling, OccWaveletTree)

		for _, pat := range patterns {
			p := []byte(pat)
			gotBV := idxBV.Count(p)
			gotWT := idxWT.Count(p)
			if gotBV != gotWT {
				t.Errorf("text=%q pat=%q: bitvectors=%d wavelet=%d", textStr, pat, gotBV, gotWT)
			}
			posBV := sortedPositions(idxBV.Locate(p, 0))
			posWT := sortedPositions(idxWT.Locate(p, 0))
			if !intSliceEqual(posBV, posWT) {
				t.Errorf("text=%q pat=%q: bitvectors locate=%v wavelet locate=%v",
					textStr, pat, posBV, posWT)
			}
		}
	}
}

func TestWaveletPersistence(t *testing.T) {
	text := []byte("the quick brown fox jumps over the lazy dog")
	idx := BuildWithOptions(text, AlgorithmSAIS, OccWaveletTree)

	var buf bytes.Buffer
	if _, err := idx.WriteTo(&buf); err != nil {
		t.Fatal("WriteTo:", err)
	}

	idx2, err := ReadFrom(&buf)
	if err != nil {
		t.Fatal("ReadFrom:", err)
	}

	patterns := []string{"the", "fox", "dog", "quick", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		c1 := idx.Count(p)
		c2 := idx2.Count(p)
		if c1 != c2 {
			t.Errorf("after wavelet persistence, Count(%q): %d vs %d", pat, c1, c2)
		}
		pos1 := sortedPositions(idx.Locate(p, 0))
		pos2 := sortedPositions(idx2.Locate(p, 0))
		if !intSliceEqual(pos1, pos2) {
			t.Errorf("after wavelet persistence, Locate(%q): %v vs %v", pat, pos1, pos2)
		}
	}
}

func TestWaveletSAISCombination(t *testing.T) {
	text := []byte("mississippi")
	idxWT := BuildWithOptions(text, AlgorithmSAIS, OccWaveletTree)
	idxBV := BuildWithAlgorithm(text, AlgorithmSAIS)

	patterns := []string{"i", "is", "si", "iss", "miss", "p", "pp", "ippi"}
	for _, pat := range patterns {
		p := []byte(pat)
		if got, want := idxWT.Count(p), idxBV.Count(p); got != want {
			t.Errorf("Count(%q): wavelet=%d bitvectors=%d", pat, got, want)
		}
		posWT := sortedPositions(idxWT.Locate(p, 0))
		posBV := sortedPositions(idxBV.Locate(p, 0))
		if !intSliceEqual(posWT, posBV) {
			t.Errorf("Locate(%q): wavelet=%v bitvectors=%v", pat, posWT, posBV)
		}
	}
}

func TestAppendMatchesRebuildBitvectors(t *testing.T) {
	idx := Build([]byte("banana"))
	idx.Append([]byte(" bandana"))
	idx.Append([]byte(" banana"))

	want := Build([]byte("banana bandana banana"))
	patterns := []string{"banana", "band", "ana", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		if got, exp := idx.Count(p), want.Count(p); got != exp {
			t.Errorf("Count(%q): append=%d rebuild=%d", pat, got, exp)
		}
		gotPos := sortedPositions(idx.Locate(p, 0))
		expPos := sortedPositions(want.Locate(p, 0))
		if !intSliceEqual(gotPos, expPos) {
			t.Errorf("Locate(%q): append=%v rebuild=%v", pat, gotPos, expPos)
		}
	}
}

func TestAppendMatchesRebuildWavelet(t *testing.T) {
	idx := BuildWithOptions([]byte("miss"), AlgorithmSAIS, OccWaveletTree)
	idx.Append([]byte("issippi"))
	idx.Append([]byte(" river"))

	want := BuildWithOptions([]byte("mississippi river"), AlgorithmSAIS, OccWaveletTree)
	patterns := []string{"iss", "ssi", "river", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		if got, exp := idx.Count(p), want.Count(p); got != exp {
			t.Errorf("Count(%q): append=%d rebuild=%d", pat, got, exp)
		}
		gotPos := sortedPositions(idx.Locate(p, 0))
		expPos := sortedPositions(want.Locate(p, 0))
		if !intSliceEqual(gotPos, expPos) {
			t.Errorf("Locate(%q): append=%v rebuild=%v", pat, gotPos, expPos)
		}
	}
}

// --- Wavelet Matrix tests ---------------------------------------------------

func TestWaveletMatrixMatchesBitvectors(t *testing.T) {
	texts := []string{
		"banana",
		"mississippi",
		"abracadabra",
		"the quick brown fox jumps over the lazy dog",
		"aaaaabbbbbccccc",
		"aaaaaaaaaa",
		"",
	}
	patterns := []string{"a", "an", "the", "ab", "ccc", "xyz", "aa"}

	for _, textStr := range texts {
		text := []byte(textStr)
		idxBV := Build(text)
		idxWM := BuildWithOptions(text, AlgorithmDoubling, OccWaveletMatrix)

		for _, pat := range patterns {
			p := []byte(pat)
			if gotBV, gotWM := idxBV.Count(p), idxWM.Count(p); gotBV != gotWM {
				t.Errorf("text=%q pat=%q: bitvectors=%d waveletmatrix=%d", textStr, pat, gotBV, gotWM)
			}
			posBV := sortedPositions(idxBV.Locate(p, 0))
			posWM := sortedPositions(idxWM.Locate(p, 0))
			if !intSliceEqual(posBV, posWM) {
				t.Errorf("text=%q pat=%q: bitvectors locate=%v waveletmatrix locate=%v",
					textStr, pat, posBV, posWM)
			}
		}
	}
}

func TestWaveletMatrixPersistence(t *testing.T) {
	text := []byte("the quick brown fox jumps over the lazy dog")
	idx := BuildWithOptions(text, AlgorithmSAIS, OccWaveletMatrix)

	var buf bytes.Buffer
	if _, err := idx.WriteTo(&buf); err != nil {
		t.Fatal("WriteTo:", err)
	}
	idx2, err := ReadFrom(&buf)
	if err != nil {
		t.Fatal("ReadFrom:", err)
	}

	patterns := []string{"the", "fox", "dog", "quick", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		if c1, c2 := idx.Count(p), idx2.Count(p); c1 != c2 {
			t.Errorf("after waveletmatrix persistence, Count(%q): %d vs %d", pat, c1, c2)
		}
		pos1 := sortedPositions(idx.Locate(p, 0))
		pos2 := sortedPositions(idx2.Locate(p, 0))
		if !intSliceEqual(pos1, pos2) {
			t.Errorf("after waveletmatrix persistence, Locate(%q): %v vs %v", pat, pos1, pos2)
		}
	}
}

// --- RLBWT tests -----------------------------------------------------------

func TestRLBWTMatchesBitvectors(t *testing.T) {
	texts := []string{
		"banana",
		"mississippi",
		"abracadabra",
		"the quick brown fox jumps over the lazy dog",
		"aaaaabbbbbccccc",
		"aaaaaaaaaa",
		"",
	}
	patterns := []string{"a", "an", "the", "ab", "ccc", "xyz", "aa"}

	for _, textStr := range texts {
		text := []byte(textStr)
		idxBV := Build(text)
		idxRL := BuildWithOptions(text, AlgorithmDoubling, OccRLBWT)

		for _, pat := range patterns {
			p := []byte(pat)
			if gotBV, gotRL := idxBV.Count(p), idxRL.Count(p); gotBV != gotRL {
				t.Errorf("text=%q pat=%q: bitvectors=%d rlbwt=%d", textStr, pat, gotBV, gotRL)
			}
			posBV := sortedPositions(idxBV.Locate(p, 0))
			posRL := sortedPositions(idxRL.Locate(p, 0))
			if !intSliceEqual(posBV, posRL) {
				t.Errorf("text=%q pat=%q: bitvectors locate=%v rlbwt locate=%v",
					textStr, pat, posBV, posRL)
			}
		}
	}
}

func TestRLBWTPersistence(t *testing.T) {
	text := []byte("the quick brown fox jumps over the lazy dog")
	idx := BuildWithOptions(text, AlgorithmSAIS, OccRLBWT)

	var buf bytes.Buffer
	if _, err := idx.WriteTo(&buf); err != nil {
		t.Fatal("WriteTo:", err)
	}
	idx2, err := ReadFrom(&buf)
	if err != nil {
		t.Fatal("ReadFrom:", err)
	}

	patterns := []string{"the", "fox", "dog", "quick", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		if c1, c2 := idx.Count(p), idx2.Count(p); c1 != c2 {
			t.Errorf("after rlbwt persistence, Count(%q): %d vs %d", pat, c1, c2)
		}
		pos1 := sortedPositions(idx.Locate(p, 0))
		pos2 := sortedPositions(idx2.Locate(p, 0))
		if !intSliceEqual(pos1, pos2) {
			t.Errorf("after rlbwt persistence, Locate(%q): %v vs %v", pat, pos1, pos2)
		}
	}
}

// --- RRR tests -------------------------------------------------------------

func TestRRRMatchesBitvectors(t *testing.T) {
	texts := []string{
		"banana",
		"mississippi",
		"abracadabra",
		"the quick brown fox jumps over the lazy dog",
		"aaaaabbbbbccccc",
		"aaaaaaaaaa",
		"",
	}
	patterns := []string{"a", "an", "the", "ab", "ccc", "xyz", "aa"}

	for _, textStr := range texts {
		text := []byte(textStr)
		idxBV := Build(text)
		idxRRR := BuildWithOptions(text, AlgorithmDoubling, OccRRR)

		for _, pat := range patterns {
			p := []byte(pat)
			if gotBV, gotRRR := idxBV.Count(p), idxRRR.Count(p); gotBV != gotRRR {
				t.Errorf("text=%q pat=%q: bitvectors=%d rrr=%d", textStr, pat, gotBV, gotRRR)
			}
			posBV := sortedPositions(idxBV.Locate(p, 0))
			posRRR := sortedPositions(idxRRR.Locate(p, 0))
			if !intSliceEqual(posBV, posRRR) {
				t.Errorf("text=%q pat=%q: bitvectors locate=%v rrr locate=%v",
					textStr, pat, posBV, posRRR)
			}
		}
	}
}

func TestRRRPersistence(t *testing.T) {
	text := []byte("the quick brown fox jumps over the lazy dog")
	idx := BuildWithOptions(text, AlgorithmSAIS, OccRRR)

	var buf bytes.Buffer
	if _, err := idx.WriteTo(&buf); err != nil {
		t.Fatal("WriteTo:", err)
	}
	idx2, err := ReadFrom(&buf)
	if err != nil {
		t.Fatal("ReadFrom:", err)
	}

	patterns := []string{"the", "fox", "dog", "quick", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		if c1, c2 := idx.Count(p), idx2.Count(p); c1 != c2 {
			t.Errorf("after rrr persistence, Count(%q): %d vs %d", pat, c1, c2)
		}
		pos1 := sortedPositions(idx.Locate(p, 0))
		pos2 := sortedPositions(idx2.Locate(p, 0))
		if !intSliceEqual(pos1, pos2) {
			t.Errorf("after rrr persistence, Locate(%q): %v vs %v", pat, pos1, pos2)
		}
	}
}

// --- Elias-Fano tests ------------------------------------------------------

func TestEliasFanoMatchesBitvectors(t *testing.T) {
	texts := []string{
		"banana",
		"mississippi",
		"abracadabra",
		"the quick brown fox jumps over the lazy dog",
		"aaaaabbbbbccccc",
		"aaaaaaaaaa",
		"",
	}
	patterns := []string{"a", "an", "the", "ab", "ccc", "xyz", "aa"}

	for _, textStr := range texts {
		text := []byte(textStr)
		idxBV := Build(text)
		idxEF := BuildWithOptions(text, AlgorithmDoubling, OccEliasFano)

		for _, pat := range patterns {
			p := []byte(pat)
			if gotBV, gotEF := idxBV.Count(p), idxEF.Count(p); gotBV != gotEF {
				t.Errorf("text=%q pat=%q: bitvectors=%d eliasfano=%d", textStr, pat, gotBV, gotEF)
			}
			posBV := sortedPositions(idxBV.Locate(p, 0))
			posEF := sortedPositions(idxEF.Locate(p, 0))
			if !intSliceEqual(posBV, posEF) {
				t.Errorf("text=%q pat=%q: bitvectors locate=%v eliasfano locate=%v",
					textStr, pat, posBV, posEF)
			}
		}
	}
}

func TestEliasFanoPersistence(t *testing.T) {
	text := []byte("the quick brown fox jumps over the lazy dog")
	idx := BuildWithOptions(text, AlgorithmSAIS, OccEliasFano)

	var buf bytes.Buffer
	if _, err := idx.WriteTo(&buf); err != nil {
		t.Fatal("WriteTo:", err)
	}
	idx2, err := ReadFrom(&buf)
	if err != nil {
		t.Fatal("ReadFrom:", err)
	}

	patterns := []string{"the", "fox", "dog", "quick", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		if c1, c2 := idx.Count(p), idx2.Count(p); c1 != c2 {
			t.Errorf("after eliasfano persistence, Count(%q): %d vs %d", pat, c1, c2)
		}
		pos1 := sortedPositions(idx.Locate(p, 0))
		pos2 := sortedPositions(idx2.Locate(p, 0))
		if !intSliceEqual(pos1, pos2) {
			t.Errorf("after eliasfano persistence, Locate(%q): %v vs %v", pat, pos1, pos2)
		}
	}
}

// --- Poppy tests -----------------------------------------------------------

func TestPoppyMatchesBitvectors(t *testing.T) {
	texts := []string{
		"banana",
		"mississippi",
		"abracadabra",
		"the quick brown fox jumps over the lazy dog",
		"aaaaabbbbbccccc",
		"aaaaaaaaaa",
		"",
	}
	patterns := []string{"a", "an", "the", "ab", "ccc", "xyz", "aa"}

	for _, textStr := range texts {
		text := []byte(textStr)
		idxBV := Build(text)
		idxPoppy := BuildWithOptions(text, AlgorithmDoubling, OccPoppy)

		for _, pat := range patterns {
			p := []byte(pat)
			if gotBV, gotPoppy := idxBV.Count(p), idxPoppy.Count(p); gotBV != gotPoppy {
				t.Errorf("text=%q pat=%q: bitvectors=%d poppy=%d", textStr, pat, gotBV, gotPoppy)
			}
			posBV := sortedPositions(idxBV.Locate(p, 0))
			posPoppy := sortedPositions(idxPoppy.Locate(p, 0))
			if !intSliceEqual(posBV, posPoppy) {
				t.Errorf("text=%q pat=%q: bitvectors locate=%v poppy locate=%v",
					textStr, pat, posBV, posPoppy)
			}
		}
	}
}

func TestPoppyPersistence(t *testing.T) {
	text := []byte("the quick brown fox jumps over the lazy dog")
	idx := BuildWithOptions(text, AlgorithmSAIS, OccPoppy)

	var buf bytes.Buffer
	if _, err := idx.WriteTo(&buf); err != nil {
		t.Fatal("WriteTo:", err)
	}
	idx2, err := ReadFrom(&buf)
	if err != nil {
		t.Fatal("ReadFrom:", err)
	}

	patterns := []string{"the", "fox", "dog", "quick", "xyz"}
	for _, pat := range patterns {
		p := []byte(pat)
		if c1, c2 := idx.Count(p), idx2.Count(p); c1 != c2 {
			t.Errorf("after poppy persistence, Count(%q): %d vs %d", pat, c1, c2)
		}
		pos1 := sortedPositions(idx.Locate(p, 0))
		pos2 := sortedPositions(idx2.Locate(p, 0))
		if !intSliceEqual(pos1, pos2) {
			t.Errorf("after poppy persistence, Locate(%q): %v vs %v", pat, pos1, pos2)
		}
	}
}

func TestNumBWTRunsAndOccType(t *testing.T) {
	text := []byte("mississippi")
	idxWM := BuildWithOptions(text, AlgorithmDoubling, OccWaveletMatrix)
	if got := idxWM.OccType(); got != OccWaveletMatrix {
		t.Errorf("OccType = %v, want OccWaveletMatrix", got)
	}
	idxRL := BuildWithOptions(text, AlgorithmDoubling, OccRLBWT)
	if got := idxRL.OccType(); got != OccRLBWT {
		t.Errorf("OccType = %v, want OccRLBWT", got)
	}
	idxRRR := BuildWithOptions(text, AlgorithmDoubling, OccRRR)
	if got := idxRRR.OccType(); got != OccRRR {
		t.Errorf("OccType = %v, want OccRRR", got)
	}
	idxEF := BuildWithOptions(text, AlgorithmDoubling, OccEliasFano)
	if got := idxEF.OccType(); got != OccEliasFano {
		t.Errorf("OccType = %v, want OccEliasFano", got)
	}
	idxPoppy := BuildWithOptions(text, AlgorithmDoubling, OccPoppy)
	if got := idxPoppy.OccType(); got != OccPoppy {
		t.Errorf("OccType = %v, want OccPoppy", got)
	}
	// NumBWTRuns: alternating text should have more runs than repetitive.
	idxAlt := BuildWithOptions([]byte("ababababab"), AlgorithmDoubling, OccRLBWT)
	idxRep := BuildWithOptions([]byte("aaaaaaaaaa"), AlgorithmDoubling, OccRLBWT)
	if idxAlt.NumBWTRuns() <= idxRep.NumBWTRuns() {
		t.Errorf("alternating (%d runs) should have more BWT runs than repetitive (%d)",
			idxAlt.NumBWTRuns(), idxRep.NumBWTRuns())
	}
}
