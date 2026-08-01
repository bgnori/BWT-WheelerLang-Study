package wavelet

import (
	"bytes"
	"testing"
)

// naiveRank counts occurrences of c in seq[0..i) by brute force.
func naiveRank(seq []byte, c byte, i int) int {
	count := 0
	for _, b := range seq[:i] {
		if b == c {
			count++
		}
	}
	return count
}

func TestRankEmpty(t *testing.T) {
	tree := Build(nil)
	for c := 0; c < 256; c++ {
		if got := tree.Rank(byte(c), 0); got != 0 {
			t.Errorf("Rank(%d, 0) on empty tree = %d, want 0", c, got)
		}
		if got := tree.Rank(byte(c), 5); got != 0 {
			t.Errorf("Rank(%d, 5) on empty tree = %d, want 0", c, got)
		}
	}
}

func TestRankSingleChar(t *testing.T) {
	seq := []byte{42}
	tree := Build(seq)

	if got := tree.Rank(42, 1); got != 1 {
		t.Errorf("Rank(42, 1) = %d, want 1", got)
	}
	if got := tree.Rank(42, 0); got != 0 {
		t.Errorf("Rank(42, 0) = %d, want 0", got)
	}
	if got := tree.Rank(43, 1); got != 0 {
		t.Errorf("Rank(43, 1) = %d, want 0", got)
	}
}

func TestRankAllSameChar(t *testing.T) {
	seq := bytes.Repeat([]byte{97}, 10) // "aaaaaaaaaa"
	tree := Build(seq)
	for i := 0; i <= 10; i++ {
		if got := tree.Rank(97, i); got != i {
			t.Errorf("Rank('a', %d) = %d, want %d", i, got, i)
		}
		if got := tree.Rank(98, i); got != 0 {
			t.Errorf("Rank('b', %d) = %d, want 0", i, got)
		}
	}
}

func TestRankMatchesNaive(t *testing.T) {
	cases := []struct {
		name string
		seq  []byte
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
			tree := Build(tc.seq)
			n := len(tc.seq)

			// Test every character value and a few positions
			testChars := []byte{'a', 'b', 'c', 'm', 'i', 's', 0, 255, 128, 'n'}
			testPos := []int{0, 1, n / 4, n / 2, 3 * n / 4, n}

			for _, c := range testChars {
				for _, i := range testPos {
					if i < 0 || i > n {
						continue
					}
					got := tree.Rank(c, i)
					want := naiveRank(tc.seq, c, i)
					if got != want {
						t.Errorf("seq=%q Rank(%d, %d) = %d, want %d", tc.name, c, i, got, want)
					}
				}
			}
		})
	}
}

func TestRankBoundaryClamp(t *testing.T) {
	seq := []byte("hello")
	tree := Build(seq)

	// i > n should be clamped to n
	want := naiveRank(seq, 'l', len(seq))
	if got := tree.Rank('l', 1000); got != want {
		t.Errorf("Rank('l', 1000) = %d, want %d", got, want)
	}
	// i = 0 should always return 0
	for _, c := range []byte("hello") {
		if got := tree.Rank(c, 0); got != 0 {
			t.Errorf("Rank(%c, 0) = %d, want 0", c, got)
		}
	}
}

func TestRankSentinelByte(t *testing.T) {
	// byte 0 (sentinel used in BWT)
	seq := []byte{'a', 0, 'b', 0, 'c'}
	tree := Build(seq)
	for i := 0; i <= len(seq); i++ {
		got := tree.Rank(0, i)
		want := naiveRank(seq, 0, i)
		if got != want {
			t.Errorf("Rank(0, %d) = %d, want %d", i, got, want)
		}
	}
}

func TestPersistence(t *testing.T) {
	seq := []byte("abracadabra")
	tree := Build(seq)

	var buf bytes.Buffer
	if _, err := tree.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	tree2, err := ReadFrom(&buf)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}

	// Verify rank results are identical
	for _, c := range []byte("abcdr") {
		for i := 0; i <= len(seq); i++ {
			got := tree2.Rank(c, i)
			want := tree.Rank(c, i)
			if got != want {
				t.Errorf("after round-trip: Rank(%c, %d) = %d, want %d", c, i, got, want)
			}
		}
	}
}

func TestPersistenceEmpty(t *testing.T) {
	tree := Build(nil)
	var buf bytes.Buffer
	if _, err := tree.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	tree2, err := ReadFrom(&buf)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}
	if got := tree2.Rank('a', 5); got != 0 {
		t.Errorf("Rank on empty tree after round-trip = %d, want 0", got)
	}
}

func TestRankFullByteRange(t *testing.T) {
	// Build a tree over all 256 byte values in order.
	seq := make([]byte, 256)
	for i := range seq {
		seq[i] = byte(i)
	}
	tree := Build(seq)
	for c := 0; c < 256; c++ {
		// c appears exactly once at position c, so Rank(c, c) = 0, Rank(c, c+1) = 1.
		if got := tree.Rank(byte(c), c); got != 0 {
			t.Errorf("Rank(%d, %d) = %d, want 0", c, c, got)
		}
		if got := tree.Rank(byte(c), c+1); got != 1 {
			t.Errorf("Rank(%d, %d) = %d, want 1", c, c+1, got)
		}
	}
}

func TestExternalRankStrategiesMatchNaive(t *testing.T) {
	seq := []byte("the quick brown fox jumps over the lazy dog")
	strategies := []struct {
		name    string
		backend ExternalBackend
	}{
		{name: "lsm", backend: ExternalBackendLSM},
		{name: "bplustree", backend: ExternalBackendBPlusTree},
		{name: "inverted", backend: ExternalBackendInvertedSegments},
	}

	for _, tc := range strategies {
		t.Run(tc.name, func(t *testing.T) {
			tree := BuildExternalWithConfig(seq, ExternalConfig{
				DiskBlockSize: 4096,
				Backend:       tc.backend,
			})
			for _, c := range []byte{'a', 'e', 'o', 'z', ' '} {
				for i := 0; i <= len(seq); i++ {
					got := tree.Rank(c, i)
					want := naiveRank(seq, c, i)
					if got != want {
						t.Fatalf("backend=%s Rank(%q, %d)=%d want=%d", tc.name, c, i, got, want)
					}
				}
			}
		})
	}
}
