package bitvector

import (
	"math/rand"
	"testing"
)

func TestRRRRankMatchesBitVector(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	const n = 1024

	bv := New(n)
	rv := NewRRR(n)
	for i := 0; i < n; i++ {
		if rng.Intn(3) == 0 {
			bv.Set(i)
			rv.Set(i)
		}
	}
	bv.Build()
	rv.Build()

	for i := 0; i <= n; i++ {
		if got, want := rv.Rank1(i), bv.Rank1(i); got != want {
			t.Fatalf("Rank1(%d) = %d, want %d", i, got, want)
		}
		if got, want := rv.Rank0(i), bv.Rank0(i); got != want {
			t.Fatalf("Rank0(%d) = %d, want %d", i, got, want)
		}
	}
}

func TestRRRGetAndTotalOnes(t *testing.T) {
	rv := NewRRR(70)
	set := map[int]bool{0: true, 1: true, 14: true, 15: true, 31: true, 32: true, 69: true}
	for i := range set {
		rv.Set(i)
	}
	rv.Build()

	for i := 0; i < 70; i++ {
		if got := rv.Get(i); got != set[i] {
			t.Fatalf("Get(%d) = %v, want %v", i, got, set[i])
		}
	}

	if got := rv.TotalOnes(); got != len(set) {
		t.Fatalf("TotalOnes() = %d, want %d", got, len(set))
	}
}

func TestPoppyRankMatchesBitVector(t *testing.T) {
	rng := rand.New(rand.NewSource(17))
	const n = 2048

	bv := New(n)
	pv := NewPoppy(n)
	for i := 0; i < n; i++ {
		if rng.Intn(4) == 0 {
			bv.Set(i)
			pv.Set(i)
		}
	}
	bv.Build()
	pv.Build()

	for i := 0; i <= n; i++ {
		if got, want := pv.Rank1(i), bv.Rank1(i); got != want {
			t.Fatalf("Rank1(%d) = %d, want %d", i, got, want)
		}
		if got, want := pv.Rank0(i), bv.Rank0(i); got != want {
			t.Fatalf("Rank0(%d) = %d, want %d", i, got, want)
		}
	}
}

func TestPoppyGetAndTotalOnes(t *testing.T) {
	pv := NewPoppy(95)
	set := map[int]bool{0: true, 2: true, 14: true, 15: true, 30: true, 31: true, 63: true, 94: true}
	for i := range set {
		pv.Set(i)
	}
	pv.Build()

	for i := 0; i < 95; i++ {
		if got := pv.Get(i); got != set[i] {
			t.Fatalf("Get(%d) = %v, want %v", i, got, set[i])
		}
	}

	if got := pv.TotalOnes(); got != len(set) {
		t.Fatalf("TotalOnes() = %d, want %d", got, len(set))
	}
}
