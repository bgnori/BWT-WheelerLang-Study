package bitvector

import "fmt"

const (
	rrrBlockSize = 15
)

var rrrBinom = buildRRRBinom(rrrBlockSize)

// RRRVector is a fixed-block Raman-Raman-Rao style bit vector.
//
// It stores each block by (class, offset) where class is the number of 1 bits
// and offset is the lexicographic rank among all bit patterns of the same
// class. Rank queries run in O(1) with a small O(blockSize) decode step for
// the last partial block.
type RRRVector struct {
	size int

	// Temporary mutable bitmap used by Set before Build.
	data []uint64

	// Per block RRR representation.
	classes []uint8
	offsets []uint32

	// prefixOnes[i] = number of 1 bits in blocks [0, i).
	prefixOnes []int
}

// NewRRR creates an empty RRRVector of size n bits.
func NewRRR(n int) *RRRVector {
	words := (n + 63) / 64
	return &RRRVector{size: n, data: make([]uint64, words)}
}

// Set sets bit i to 1.
func (rv *RRRVector) Set(i int) {
	rv.data[i/64] |= 1 << uint(i%64)
}

// Get returns the value of bit i.
func (rv *RRRVector) Get(i int) bool {
	if i < 0 || i >= rv.size {
		return false
	}
	return rv.Rank1(i+1)-rv.Rank1(i) == 1
}

// Build constructs the RRR block representation and rank prefixes.
func (rv *RRRVector) Build() {
	if rv.size == 0 {
		rv.classes = nil
		rv.offsets = nil
		rv.prefixOnes = []int{0}
		rv.data = nil
		return
	}

	nBlocks := (rv.size + rrrBlockSize - 1) / rrrBlockSize
	rv.classes = make([]uint8, nBlocks)
	rv.offsets = make([]uint32, nBlocks)
	rv.prefixOnes = make([]int, nBlocks+1)

	for block := 0; block < nBlocks; block++ {
		start := block * rrrBlockSize
		blockLen := rrrBlockSize
		if remain := rv.size - start; remain < blockLen {
			blockLen = remain
		}

		bits := uint16(0)
		for j := 0; j < blockLen; j++ {
			idx := start + j
			if (rv.data[idx/64]>>uint(idx%64))&1 == 1 {
				bits |= 1 << uint(j)
			}
		}

		class, offset := rrrEncode(bits, blockLen)
		rv.classes[block] = uint8(class)
		rv.offsets[block] = uint32(offset)
		rv.prefixOnes[block+1] = rv.prefixOnes[block] + class
	}

	// Raw words are no longer needed after the compact representation is built.
	rv.data = nil
}

// Rank1 returns the number of 1 bits in [0, i).
func (rv *RRRVector) Rank1(i int) int {
	if i <= 0 {
		return 0
	}
	if i > rv.size {
		i = rv.size
	}
	if rv.size == 0 {
		return 0
	}

	block := i / rrrBlockSize
	bit := i % rrrBlockSize
	if block >= len(rv.classes) {
		return rv.prefixOnes[len(rv.prefixOnes)-1]
	}

	count := rv.prefixOnes[block]
	if bit > 0 {
		start := block * rrrBlockSize
		blockLen := rrrBlockSize
		if remain := rv.size - start; remain < blockLen {
			blockLen = remain
		}
		count += rrrRankPrefix(int(rv.classes[block]), int(rv.offsets[block]), blockLen, bit)
	}
	return count
}

// Rank0 returns the number of 0 bits in [0, i).
func (rv *RRRVector) Rank0(i int) int {
	if i <= 0 {
		return 0
	}
	if i > rv.size {
		i = rv.size
	}
	return i - rv.Rank1(i)
}

// Size returns the number of bits in the vector.
func (rv *RRRVector) Size() int { return rv.size }

// TotalOnes returns the total number of 1 bits.
func (rv *RRRVector) TotalOnes() int {
	if len(rv.prefixOnes) == 0 {
		return 0
	}
	return rv.prefixOnes[len(rv.prefixOnes)-1]
}

func buildRRRBinom(maxN int) [][]int {
	binom := make([][]int, maxN+1)
	for n := 0; n <= maxN; n++ {
		binom[n] = make([]int, n+1)
		binom[n][0] = 1
		binom[n][n] = 1
		for k := 1; k < n; k++ {
			binom[n][k] = binom[n-1][k-1] + binom[n-1][k]
		}
	}
	return binom
}

func rrrChoose(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	return rrrBinom[n][k]
}

// rrrEncode returns (class, offset) for the given block bits.
func rrrEncode(bits uint16, blockLen int) (int, int) {
	if blockLen < 0 || blockLen > rrrBlockSize {
		panic(fmt.Sprintf("bitvector: invalid RRR block length %d", blockLen))
	}
	class := popcount(uint64(bits))
	offset := 0
	onesLeft := class

	for pos := 0; pos < blockLen; pos++ {
		if onesLeft == 0 {
			break
		}
		if (bits>>uint(pos))&1 == 1 {
			rem := blockLen - pos - 1
			offset += rrrChoose(rem, onesLeft)
			onesLeft--
		}
	}
	return class, offset
}

// rrrRankPrefix returns the number of 1 bits in the first prefixLen positions
// of a block represented by (class, offset).
func rrrRankPrefix(class, offset, blockLen, prefixLen int) int {
	if prefixLen <= 0 || class == 0 {
		return 0
	}
	if prefixLen > blockLen {
		prefixLen = blockLen
	}

	ones := 0
	onesLeft := class
	code := offset

	for pos := 0; pos < prefixLen; pos++ {
		if onesLeft == 0 {
			break
		}
		rem := blockLen - pos - 1
		zeroCount := rrrChoose(rem, onesLeft)
		if code < zeroCount {
			continue
		}
		ones++
		code -= zeroCount
		onesLeft--
	}
	return ones
}
