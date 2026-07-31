package rindex

import (
	"bytes"
	"testing"
)

// naiveRank counts occurrences of b in bwt[0..i) by brute force.
func naiveRank(bwt []byte, b byte, i int) int {
	count := 0
	for _, c := range bwt[:i] {
		if c == b {
			count++
		}
	}
	return count
}

func TestRankEmpty(t *testing.T) {
	rl := Build(nil)
	for c := 0; c < 256; c++ {
		if got := rl.Rank(byte(c), 0); got != 0 {
			t.Errorf("Rank(%d, 0) on empty = %d, want 0", c, got)
		}
		if got := rl.Rank(byte(c), 5); got != 0 {
			t.Errorf("Rank(%d, 5) on empty = %d, want 0", c, got)
		}
	}
}

func TestRankSingleChar(t *testing.T) {
	bwt := []byte{42}
	rl := Build(bwt)

	if got := rl.Rank(42, 1); got != 1 {
		t.Errorf("Rank(42, 1) = %d, want 1", got)
	}
	if got := rl.Rank(42, 0); got != 0 {
		t.Errorf("Rank(42, 0) = %d, want 0", got)
	}
	if got := rl.Rank(43, 1); got != 0 {
		t.Errorf("Rank(43, 1) = %d, want 0", got)
	}
}

func TestRankAllSameChar(t *testing.T) {
	bwt := bytes.Repeat([]byte{97}, 10)
	rl := Build(bwt)
	if got := rl.NumRuns(); got != 1 {
		t.Errorf("NumRuns() = %d, want 1 (all same char)", got)
	}
	for i := 0; i <= 10; i++ {
		if got := rl.Rank(97, i); got != i {
			t.Errorf("Rank('a', %d) = %d, want %d", i, got, i)
		}
		if got := rl.Rank(98, i); got != 0 {
			t.Errorf("Rank('b', %d) = %d, want 0", i, got)
		}
	}
}

func TestRankMatchesNaive(t *testing.T) {
	cases := []struct {
		name string
		bwt  []byte
	}{
		{"banana", []byte("banana")},
		{"mississippi", []byte("mississippi")},
		{"abracadabra", []byte("abracadabra")},
		{"the quick brown fox", []byte("the quick brown fox")},
		{"all_bytes", func() []byte {
			b := make([]byte, 256)
			for i := range b {
				b[i] = byte(i)
			}
			return b
		}()},
		{"repeated_pattern", bytes.Repeat([]byte("abc"), 20)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rl := Build(tc.bwt)
			n := len(tc.bwt)

			testChars := []byte{'a', 'b', 'c', 'm', 'i', 's', 0, 255, 128, 'n'}
			testPos := []int{0, 1, n / 4, n / 2, 3 * n / 4, n}

			for _, c := range testChars {
				for _, pos := range testPos {
					if pos < 0 || pos > n {
						continue
					}
					got := rl.Rank(c, pos)
					want := naiveRank(tc.bwt, c, pos)
					if got != want {
						t.Errorf("bwt=%q Rank(%d, %d) = %d, want %d", tc.name, c, pos, got, want)
					}
				}
			}
		})
	}
}

func TestRankBoundaryClamp(t *testing.T) {
	bwt := []byte("hello")
	rl := Build(bwt)

	want := naiveRank(bwt, 'l', len(bwt))
	if got := rl.Rank('l', 1000); got != want {
		t.Errorf("Rank('l', 1000) = %d, want %d (clamped)", got, want)
	}
	for _, c := range []byte("hello") {
		if got := rl.Rank(c, 0); got != 0 {
			t.Errorf("Rank(%c, 0) = %d, want 0", c, got)
		}
	}
}

func TestRankSentinelByte(t *testing.T) {
	bwt := []byte{'a', 0, 'b', 0, 'c'}
	rl := Build(bwt)
	for i := 0; i <= len(bwt); i++ {
		got := rl.Rank(0, i)
		want := naiveRank(bwt, 0, i)
		if got != want {
			t.Errorf("Rank(0, %d) = %d, want %d", i, got, want)
		}
	}
}

func TestNumRunsCompression(t *testing.T) {
	// "aaabbbccc" has 3 runs.
	bwt := []byte("aaabbbccc")
	rl := Build(bwt)
	if got := rl.NumRuns(); got != 3 {
		t.Errorf("NumRuns() = %d, want 3", got)
	}
	// "abcabc" has 6 runs (alternating chars).
	rl2 := Build([]byte("abcabc"))
	if got := rl2.NumRuns(); got != 6 {
		t.Errorf("NumRuns() for alternating = %d, want 6", got)
	}
}

func TestPersistence(t *testing.T) {
	bwt := []byte("abracadabra")
	rl := Build(bwt)

	var buf bytes.Buffer
	if _, err := rl.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	rl2, err := ReadFrom(&buf)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}

	for _, c := range []byte("abcdr") {
		for i := 0; i <= len(bwt); i++ {
			got := rl2.Rank(c, i)
			want := rl.Rank(c, i)
			if got != want {
				t.Errorf("after round-trip: Rank(%c, %d) = %d, want %d", c, i, got, want)
			}
		}
	}
}

func TestMatchesNaiveExhaustive(t *testing.T) {
	bwt := []byte("mississippi")
	rl := Build(bwt)
	n := len(bwt)
	for c := 0; c < 256; c++ {
		for i := 0; i <= n; i++ {
			got := rl.Rank(byte(c), i)
			want := naiveRank(bwt, byte(c), i)
			if got != want {
				t.Errorf("Rank(%d, %d) = %d, want %d", c, i, got, want)
			}
		}
	}
}
