// Package bitvector provides a succinct bit vector with O(1) Rank1/Rank0 queries.
// It uses a flat block structure: block[i] stores the cumulative count of 1-bits
// in all 64-bit words before index i, enabling fast rank queries via popcount.
package bitvector

import (
	"encoding/binary"
	"fmt"
	"io"
)

// BitVector is a compact bit array with O(1) rank support.
type BitVector struct {
	data  []uint64
	size  int
	block []int // block[i] = # of 1-bits in data[0..i-1] (first i*64 bits)
}

// New creates a BitVector of size n bits, all zero.
// Call Build after all Set operations.
func New(n int) *BitVector {
	words := (n + 63) / 64
	return &BitVector{data: make([]uint64, words), size: n}
}

// Set sets bit i to 1.
func (bv *BitVector) Set(i int) {
	bv.data[i/64] |= 1 << uint(i%64)
}

// Get returns the value of bit i.
func (bv *BitVector) Get(i int) bool {
	return (bv.data[i/64]>>uint(i%64))&1 == 1
}

// Build computes the rank index. Must be called after all Set calls
// and before any Rank1/Rank0 queries.
func (bv *BitVector) Build() {
	bv.block = make([]int, len(bv.data)+1)
	for i, w := range bv.data {
		bv.block[i+1] = bv.block[i] + popcount(w)
	}
}

// Rank1 returns the number of 1-bits in positions [0, i).
func (bv *BitVector) Rank1(i int) int {
	if i <= 0 {
		return 0
	}
	if i > bv.size {
		i = bv.size
	}
	word := i / 64
	bit := uint(i % 64)
	count := bv.block[word]
	if bit > 0 {
		count += popcount(bv.data[word] & ((1 << bit) - 1))
	}
	return count
}

// Rank0 returns the number of 0-bits in positions [0, i).
func (bv *BitVector) Rank0(i int) int {
	return i - bv.Rank1(i)
}

// Size returns the number of bits in the vector.
func (bv *BitVector) Size() int { return bv.size }

// TotalOnes returns the total number of 1-bits.
func (bv *BitVector) TotalOnes() int {
	if len(bv.block) == 0 {
		return 0
	}
	return bv.block[len(bv.block)-1]
}

// popcount counts the 1-bits in a 64-bit word using the Hamming weight algorithm.
func popcount(x uint64) int {
	x -= (x >> 1) & 0x5555555555555555
	x = (x & 0x3333333333333333) + ((x >> 2) & 0x3333333333333333)
	x = (x + (x >> 4)) & 0x0f0f0f0f0f0f0f0f
	return int((x * 0x0101010101010101) >> 56)
}

// WriteTo serialises the BitVector to w in little-endian binary format.
// It implements io.WriterTo.
func (bv *BitVector) WriteTo(w io.Writer) (int64, error) {
	var written int64

	if err := binary.Write(w, binary.LittleEndian, int64(bv.size)); err != nil {
		return written, fmt.Errorf("bitvector: write size: %w", err)
	}
	written += 8

	nwords := int64(len(bv.data))
	if err := binary.Write(w, binary.LittleEndian, nwords); err != nil {
		return written, fmt.Errorf("bitvector: write nwords: %w", err)
	}
	written += 8

	if nwords > 0 {
		buf := make([]byte, nwords*8)
		for i, d := range bv.data {
			binary.LittleEndian.PutUint64(buf[i*8:], d)
		}
		n, err := w.Write(buf)
		written += int64(n)
		if err != nil {
			return written, fmt.Errorf("bitvector: write data: %w", err)
		}
	}
	return written, nil
}

// ReadFrom deserialises a BitVector from r and rebuilds the rank index.
func ReadFrom(r io.Reader) (*BitVector, error) {
	var size, nwords int64
	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return nil, fmt.Errorf("bitvector: read size: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &nwords); err != nil {
		return nil, fmt.Errorf("bitvector: read nwords: %w", err)
	}
	data := make([]uint64, nwords)
	if nwords > 0 {
		buf := make([]byte, nwords*8)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, fmt.Errorf("bitvector: read data: %w", err)
		}
		for i := range data {
			data[i] = binary.LittleEndian.Uint64(buf[i*8:])
		}
	}
	bv := &BitVector{data: data, size: int(size)}
	bv.Build()
	return bv, nil
}
