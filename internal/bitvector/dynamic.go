package bitvector

// DynamicBitVector is a mutable bit vector that supports rank queries.
//
// It stores bits in packed 64-bit words and maintains a Fenwick tree over
// per-word popcounts to answer Rank1 in O(log W), where W is the number of
// words. Point updates run in O(log W). Insert/Delete rebuild the Fenwick
// index after shifting bits.
type DynamicBitVector struct {
	data []uint64
	size int
	ft   []int
}

// NewDynamic creates a DynamicBitVector with n zero bits.
func NewDynamic(n int) *DynamicBitVector {
	if n < 0 {
		n = 0
	}
	words := (n + 63) / 64
	dv := &DynamicBitVector{
		data: make([]uint64, words),
		size: n,
	}
	dv.rebuildFenwick()
	return dv
}

// Size returns the number of bits in the vector.
func (dv *DynamicBitVector) Size() int { return dv.size }

// Build rebuilds rank metadata.
func (dv *DynamicBitVector) Build() { dv.rebuildFenwick() }

// Get returns the value of bit i.
func (dv *DynamicBitVector) Get(i int) bool {
	if i < 0 || i >= dv.size {
		return false
	}
	return dv.bitRaw(i)
}

// Set sets bit i to 1.
func (dv *DynamicBitVector) Set(i int) {
	if i < 0 || i >= dv.size {
		return
	}
	if dv.bitRaw(i) {
		return
	}
	dv.setRaw(i, true)
	dv.fenwickAdd(i/64+1, 1)
}

// Clear sets bit i to 0.
func (dv *DynamicBitVector) Clear(i int) {
	if i < 0 || i >= dv.size {
		return
	}
	if !dv.bitRaw(i) {
		return
	}
	dv.setRaw(i, false)
	dv.fenwickAdd(i/64+1, -1)
}

// Insert inserts one bit before index i.
// Valid range is 0 <= i <= Size().
func (dv *DynamicBitVector) Insert(i int, value bool) {
	if i < 0 || i > dv.size {
		return
	}

	oldSize := dv.size
	dv.size++
	dv.ensureWords((dv.size + 63) / 64)

	for pos := oldSize; pos > i; pos-- {
		dv.setRaw(pos, dv.bitRaw(pos-1))
	}
	dv.setRaw(i, value)
	dv.rebuildFenwick()
}

// Delete removes and returns the bit at index i.
func (dv *DynamicBitVector) Delete(i int) bool {
	if i < 0 || i >= dv.size {
		return false
	}

	removed := dv.bitRaw(i)
	for pos := i; pos < dv.size-1; pos++ {
		dv.setRaw(pos, dv.bitRaw(pos+1))
	}

	last := dv.size - 1
	dv.setRaw(last, false)
	dv.size--
	dv.trimWords()
	dv.rebuildFenwick()
	return removed
}

// Rank1 returns the number of 1-bits in [0, i).
func (dv *DynamicBitVector) Rank1(i int) int {
	if i <= 0 || dv.size == 0 {
		return 0
	}
	if i > dv.size {
		i = dv.size
	}

	word := i / 64
	bit := uint(i % 64)
	count := dv.fenwickPrefix(word)
	if bit > 0 && word < len(dv.data) {
		count += popcount(dv.data[word] & ((uint64(1) << bit) - 1))
	}
	return count
}

// Rank0 returns the number of 0-bits in [0, i).
func (dv *DynamicBitVector) Rank0(i int) int {
	if i <= 0 {
		return 0
	}
	if i > dv.size {
		i = dv.size
	}
	return i - dv.Rank1(i)
}

// TotalOnes returns the total number of 1-bits.
func (dv *DynamicBitVector) TotalOnes() int {
	return dv.Rank1(dv.size)
}

func (dv *DynamicBitVector) bitRaw(i int) bool {
	return (dv.data[i/64]>>uint(i%64))&1 == 1
}

func (dv *DynamicBitVector) setRaw(i int, value bool) {
	word := i / 64
	mask := uint64(1) << uint(i%64)
	if value {
		dv.data[word] |= mask
		return
	}
	dv.data[word] &^= mask
}

func (dv *DynamicBitVector) ensureWords(words int) {
	if words <= len(dv.data) {
		return
	}
	extra := make([]uint64, words-len(dv.data))
	dv.data = append(dv.data, extra...)
}

func (dv *DynamicBitVector) trimWords() {
	need := (dv.size + 63) / 64
	if need == 0 {
		dv.data = nil
		return
	}
	if len(dv.data) > need {
		dv.data = dv.data[:need]
	}
	used := dv.size % 64
	if used != 0 {
		mask := (uint64(1) << uint(used)) - 1
		dv.data[need-1] &= mask
	}
}

func (dv *DynamicBitVector) rebuildFenwick() {
	dv.ft = make([]int, len(dv.data)+1)
	for i, w := range dv.data {
		dv.fenwickAdd(i+1, popcount(w))
	}
}

func (dv *DynamicBitVector) fenwickAdd(i, delta int) {
	for i < len(dv.ft) {
		dv.ft[i] += delta
		i += i & -i
	}
}

// fenwickPrefix returns the sum over the first i words.
func (dv *DynamicBitVector) fenwickPrefix(i int) int {
	if i <= 0 {
		return 0
	}
	if i >= len(dv.ft) {
		i = len(dv.ft) - 1
	}
	sum := 0
	for i > 0 {
		sum += dv.ft[i]
		i -= i & -i
	}
	return sum
}
