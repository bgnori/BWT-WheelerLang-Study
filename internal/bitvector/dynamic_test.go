package bitvector

import "testing"

func TestDynamicBitVectorRankAndGet(t *testing.T) {
	dv := NewDynamic(130)
	set := map[int]bool{0: true, 1: true, 63: true, 64: true, 65: true, 129: true}
	for i := range set {
		dv.Set(i)
	}

	for i := 0; i < dv.Size(); i++ {
		if got := dv.Get(i); got != set[i] {
			t.Fatalf("Get(%d) = %v, want %v", i, got, set[i])
		}
	}

	cases := []struct {
		i    int
		want int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{63, 2},
		{64, 3},
		{65, 4},
		{66, 5},
		{130, 6},
	}
	for _, tc := range cases {
		if got := dv.Rank1(tc.i); got != tc.want {
			t.Fatalf("Rank1(%d) = %d, want %d", tc.i, got, tc.want)
		}
	}
}

func TestDynamicBitVectorInsertDelete(t *testing.T) {
	dv := NewDynamic(0)
	var naive []bool

	insert := func(pos int, value bool) {
		if pos < 0 || pos > len(naive) {
			t.Fatalf("invalid insert pos %d for naive size %d", pos, len(naive))
		}
		naive = append(naive, false)
		copy(naive[pos+1:], naive[pos:])
		naive[pos] = value
		dv.Insert(pos, value)
	}

	deleteAt := func(pos int) {
		if pos < 0 || pos >= len(naive) {
			t.Fatalf("invalid delete pos %d for naive size %d", pos, len(naive))
		}
		want := naive[pos]
		got := dv.Delete(pos)
		if got != want {
			t.Fatalf("Delete(%d) returned %v, want %v", pos, got, want)
		}
		copy(naive[pos:], naive[pos+1:])
		naive = naive[:len(naive)-1]
	}

	insert(0, true)
	insert(1, false)
	insert(1, true)
	insert(3, true)
	insert(2, false)
	deleteAt(1)
	insert(2, true)
	deleteAt(0)

	if dv.Size() != len(naive) {
		t.Fatalf("Size() = %d, want %d", dv.Size(), len(naive))
	}

	prefix := 0
	for i := 0; i < len(naive); i++ {
		if got := dv.Get(i); got != naive[i] {
			t.Fatalf("Get(%d) = %v, want %v", i, got, naive[i])
		}
		if naive[i] {
			prefix++
		}
		if got := dv.Rank1(i + 1); got != prefix {
			t.Fatalf("Rank1(%d) = %d, want %d", i+1, got, prefix)
		}
	}
}
