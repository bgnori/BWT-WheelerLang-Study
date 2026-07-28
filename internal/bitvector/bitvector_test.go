package bitvector

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestGetSet(t *testing.T) {
	bv := New(200)
	bv.Set(0)
	bv.Set(1)
	bv.Set(63)
	bv.Set(64)
	bv.Set(65)
	bv.Set(127)
	bv.Set(128)
	bv.Set(199)
	bv.Build()

	set := map[int]bool{0: true, 1: true, 63: true, 64: true, 65: true, 127: true, 128: true, 199: true}
	for i := 0; i < 200; i++ {
		if got := bv.Get(i); got != set[i] {
			t.Errorf("Get(%d) = %v, want %v", i, got, set[i])
		}
	}
}

func TestRank1(t *testing.T) {
	bv := New(200)
	// Set bits: 0, 1, 63, 64, 99
	bv.Set(0)
	bv.Set(1)
	bv.Set(63)
	bv.Set(64)
	bv.Set(99)
	bv.Build()

	cases := []struct {
		i    int
		want int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 2},
		{63, 2},
		{64, 3},
		{65, 4},
		{66, 4},
		{99, 4},
		{100, 5},
		{200, 5},
	}
	for _, tc := range cases {
		if got := bv.Rank1(tc.i); got != tc.want {
			t.Errorf("Rank1(%d) = %d, want %d", tc.i, got, tc.want)
		}
	}
}

func TestRank0(t *testing.T) {
	bv := New(100)
	bv.Set(0)
	bv.Set(2)
	bv.Set(4)
	bv.Build()

	// Rank0(5) = 5 - Rank1(5) = 5 - 3 = 2
	if got := bv.Rank0(5); got != 2 {
		t.Errorf("Rank0(5) = %d, want 2", got)
	}
}

func TestRank1BoundaryWordEdge(t *testing.T) {
	// All 64 bits set in the first word
	bv := New(128)
	for i := 0; i < 64; i++ {
		bv.Set(i)
	}
	bv.Build()

	if got := bv.Rank1(64); got != 64 {
		t.Errorf("Rank1(64) = %d, want 64", got)
	}
	if got := bv.Rank1(65); got != 64 {
		t.Errorf("Rank1(65) = %d, want 64", got)
	}
	if got := bv.Rank1(0); got != 0 {
		t.Errorf("Rank1(0) = %d, want 0", got)
	}
}

func TestRoundTrip(t *testing.T) {
	bv := New(500)
	for i := 0; i < 500; i += 3 {
		bv.Set(i)
	}
	bv.Build()

	var buf bytes.Buffer
	if _, err := bv.WriteTo(&buf); err != nil {
		t.Fatal("WriteTo:", err)
	}
	bv2, err := ReadFrom(&buf)
	if err != nil {
		t.Fatal("ReadFrom:", err)
	}

	for i := 0; i <= 500; i++ {
		if bv.Rank1(i) != bv2.Rank1(i) {
			t.Errorf("Rank1(%d) mismatch after round-trip: %d vs %d", i, bv.Rank1(i), bv2.Rank1(i))
		}
	}
	for i := 0; i < 500; i++ {
		if bv.Get(i) != bv2.Get(i) {
			t.Errorf("Get(%d) mismatch after round-trip", i)
		}
	}
}

func TestRank1Random(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	n := 1000
	bv := New(n)
	var naive []int // naive[i] = # of 1-bits in [0,i)
	bits := make([]bool, n)

	for i := 0; i < n; i++ {
		if rng.Intn(2) == 1 {
			bv.Set(i)
			bits[i] = true
		}
	}
	bv.Build()

	// Build naive prefix sums
	naive = make([]int, n+1)
	for i := 0; i < n; i++ {
		naive[i+1] = naive[i]
		if bits[i] {
			naive[i+1]++
		}
	}

	for i := 0; i <= n; i++ {
		if got := bv.Rank1(i); got != naive[i] {
			t.Errorf("Rank1(%d) = %d, want %d", i, got, naive[i])
		}
	}
}

func TestTotalOnes(t *testing.T) {
	bv := New(100)
	for i := 0; i < 100; i += 2 {
		bv.Set(i)
	}
	bv.Build()
	if got := bv.TotalOnes(); got != 50 {
		t.Errorf("TotalOnes() = %d, want 50", got)
	}
}
