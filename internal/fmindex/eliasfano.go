package fmindex

import "math/bits"

// eliasFano stores a sorted integer sequence in [0, universe) using an
// Elias-Fano style split into upper groups and packed lower bits.
type eliasFano struct {
	n          int
	universe   int
	lowerBits  uint
	lowerMask  uint64
	lowerWords []uint64
	groupStart []int // start offsets by upper value; len = upperMax+2
}

func newEliasFano(sorted []int, universe int) *eliasFano {
	ef := &eliasFano{n: len(sorted), universe: universe}
	if ef.n == 0 || universe <= 0 {
		ef.groupStart = []int{0, 0}
		return ef
	}

	ratio := universe / ef.n
	if ratio > 1 {
		ef.lowerBits = uint(bits.Len(uint(ratio)) - 1)
		if ef.lowerBits > 63 {
			ef.lowerBits = 63
		}
	}
	if ef.lowerBits > 0 {
		ef.lowerMask = (uint64(1) << ef.lowerBits) - 1
	}

	upperMax := (universe - 1) >> ef.lowerBits
	counts := make([]int, upperMax+1)
	for _, v := range sorted {
		h := v >> ef.lowerBits
		counts[h]++
	}
	ef.groupStart = make([]int, upperMax+2)
	for h := 0; h <= upperMax; h++ {
		ef.groupStart[h+1] = ef.groupStart[h] + counts[h]
	}

	if ef.lowerBits == 0 {
		return ef
	}

	totalLowerBits := ef.n * int(ef.lowerBits)
	ef.lowerWords = make([]uint64, (totalLowerBits+63)/64)
	for i, v := range sorted {
		ef.setLower(i, uint64(v)&ef.lowerMask)
	}
	return ef
}

func (ef *eliasFano) setLower(i int, value uint64) {
	bitPos := i * int(ef.lowerBits)
	w := bitPos / 64
	off := uint(bitPos % 64)
	ef.lowerWords[w] |= value << off
	if off+ef.lowerBits > 64 {
		ef.lowerWords[w+1] |= value >> (64 - off)
	}
}

func (ef *eliasFano) lowerAt(i int) uint64 {
	if ef.lowerBits == 0 {
		return 0
	}
	bitPos := i * int(ef.lowerBits)
	w := bitPos / 64
	off := uint(bitPos % 64)
	v := ef.lowerWords[w] >> off
	if off+ef.lowerBits > 64 {
		v |= ef.lowerWords[w+1] << (64 - off)
	}
	return v & ef.lowerMask
}

// rankLessThan returns the number of stored values strictly less than x.
func (ef *eliasFano) rankLessThan(x int) int {
	if ef.n == 0 || x <= 0 {
		return 0
	}
	if x >= ef.universe {
		return ef.n
	}

	h := x >> ef.lowerBits
	start := ef.groupStart[h]
	end := ef.groupStart[h+1]
	if ef.lowerBits == 0 || start == end {
		return start
	}

	targetLow := uint64(x) & ef.lowerMask
	lo, hi := start, end
	for lo < hi {
		mid := lo + (hi-lo)/2
		if ef.lowerAt(mid) < targetLow {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// eliasFanoOcc implements occStructure using one Elias-Fano sequence per byte.
type eliasFanoOcc struct {
	sets [256]*eliasFano
}

func buildEliasFanoOcc(bwt []byte) *eliasFanoOcc {
	occ := &eliasFanoOcc{}
	var positions [256][]int
	for i, b := range bwt {
		positions[b] = append(positions[b], i)
	}
	n := len(bwt)
	for c := 0; c < 256; c++ {
		if len(positions[c]) == 0 {
			continue
		}
		occ.sets[c] = newEliasFano(positions[c], n)
	}
	return occ
}

func (o *eliasFanoOcc) rank(b byte, i int) int {
	if o.sets[b] == nil {
		return 0
	}
	return o.sets[b].rankLessThan(i)
}
