// Package waveletmatrix provides a Wavelet Matrix for rank queries over byte sequences.
//
// A Wavelet Matrix is a flat, level-by-level alternative to a Wavelet Tree.
// It stores log2(256) = 8 bit-vectors, one per bit position of the byte alphabet.
// At each level the current sequence is stably partitioned so that symbols with
// bit = 0 come before symbols with bit = 1.  Rank queries run in O(8) time.
//
// Unlike a Wavelet Tree, which allocates separate bit-vectors per subtree node,
// the Wavelet Matrix uses a flat layout where all levels live side-by-side,
// making it more cache-friendly in practice.
package waveletmatrix

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/bgnori/bwt-wheelerlang-study/internal/bitvector"
)

const levels = 8 // log2(256) for byte alphabet

// Matrix is a Wavelet Matrix built over a byte sequence.
type Matrix struct {
	n      int
	bvs    [levels]*bitvector.BitVector // one bit-vector per level (level 0 = MSB)
	nzeros [levels]int                  // nzeros[l] = number of 0-bits in bvs[l]
}

// Build constructs a Wavelet Matrix from the byte sequence seq.
func Build(seq []byte) *Matrix {
	n := len(seq)
	m := &Matrix{n: n}
	if n == 0 {
		for l := 0; l < levels; l++ {
			m.bvs[l] = bitvector.New(0)
			m.bvs[l].Build()
		}
		return m
	}

	cur := make([]byte, n)
	copy(cur, seq)

	// Process from the most-significant bit (l=7) down to the least-significant
	// bit (l=0).  At each iteration l is the bit index: (c >> l) & 1 gives the
	// bit for that level.
	for l := levels - 1; l >= 0; l-- {
		bv := bitvector.New(n)
		zeros := make([]byte, 0, n)
		ones := make([]byte, 0, n)
		for i, c := range cur {
			if (c>>uint(l))&1 == 1 {
				bv.Set(i)
				ones = append(ones, c)
			} else {
				zeros = append(zeros, c)
			}
		}
		bv.Build()
		m.bvs[l] = bv
		m.nzeros[l] = len(zeros)
		// Stable partition for the next level: zeros first, ones after.
		copy(cur, zeros)
		copy(cur[len(zeros):], ones)
	}
	return m
}

// Rank returns the number of occurrences of c in seq[0..i).
// i is clamped to [0, n].
func (m *Matrix) Rank(c byte, i int) int {
	if m.n == 0 || i <= 0 {
		return 0
	}
	if i > m.n {
		i = m.n
	}

	// Track the half-open range [lo, hi) in the current level's virtual sequence.
	// Initially [0, i) in the original sequence.
	// After all 8 levels, hi - lo equals the count of c in seq[0..i).
	lo := 0
	hi := i

	for l := levels - 1; l >= 0; l-- {
		bv := m.bvs[l]
		if (c>>uint(l))&1 == 0 {
			// Navigate left: keep only the 0-bits in [lo, hi).
			lo = lo - bv.Rank1(lo)
			hi = hi - bv.Rank1(hi)
		} else {
			// Navigate right: map into the right partition using nzeros offset.
			lo = m.nzeros[l] + bv.Rank1(lo)
			hi = m.nzeros[l] + bv.Rank1(hi)
		}
	}
	return hi - lo
}

// WriteTo serialises the Wavelet Matrix to w.
// It implements io.WriterTo.
func (m *Matrix) WriteTo(w io.Writer) (int64, error) {
	var written int64
	if err := binary.Write(w, binary.LittleEndian, int64(m.n)); err != nil {
		return 0, fmt.Errorf("waveletmatrix: write n: %w", err)
	}
	written += 8
	for l := 0; l < levels; l++ {
		if err := binary.Write(w, binary.LittleEndian, int64(m.nzeros[l])); err != nil {
			return written, fmt.Errorf("waveletmatrix: write nzeros[%d]: %w", l, err)
		}
		written += 8
		n, err := m.bvs[l].WriteTo(w)
		written += n
		if err != nil {
			return written, fmt.Errorf("waveletmatrix: write bv[%d]: %w", l, err)
		}
	}
	return written, nil
}

// ReadFrom deserialises a Wavelet Matrix from r.
func ReadFrom(r io.Reader) (*Matrix, error) {
	var n64 int64
	if err := binary.Read(r, binary.LittleEndian, &n64); err != nil {
		return nil, fmt.Errorf("waveletmatrix: read n: %w", err)
	}
	m := &Matrix{n: int(n64)}
	for l := 0; l < levels; l++ {
		var nz int64
		if err := binary.Read(r, binary.LittleEndian, &nz); err != nil {
			return nil, fmt.Errorf("waveletmatrix: read nzeros[%d]: %w", l, err)
		}
		m.nzeros[l] = int(nz)
		bv, err := bitvector.ReadFrom(r)
		if err != nil {
			return nil, fmt.Errorf("waveletmatrix: read bv[%d]: %w", l, err)
		}
		m.bvs[l] = bv
	}
	return m, nil
}
