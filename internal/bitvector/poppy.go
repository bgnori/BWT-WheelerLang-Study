package bitvector

const (
	poppyBlockSize      = rrrBlockSize
	poppyBlocksPerSuper = 32
)

// PoppyVector is an interleaved RRR-style bit vector.
//
// It stores each block as a packed (class, offset) entry and keeps two-level
// rank samples: superblock prefixes and per-block prefixes within each
// superblock. This layout follows the "Poppy" idea of keeping local metadata
// close to compressed block data for cache-friendly rank queries.
type PoppyVector struct {
	size int

	// Temporary mutable bitmap used by Set before Build.
	data []uint64

	// Per block packed RRR representation:
	// low 8 bits: class (number of ones), high bits: offset.
	entries []uint32

	// superPrefix[s] = number of ones in all blocks before superblock s.
	superPrefix []int

	// blockPrefix[b] = number of ones in the current superblock before block b.
	blockPrefix []uint16
}

// NewPoppy creates an empty interleaved-RRR (Poppy) vector of size n bits.
func NewPoppy(n int) *PoppyVector {
	words := (n + 63) / 64
	return &PoppyVector{size: n, data: make([]uint64, words)}
}

// Set sets bit i to 1.
func (pv *PoppyVector) Set(i int) {
	pv.data[i/64] |= 1 << uint(i%64)
}

// Get returns the value of bit i.
func (pv *PoppyVector) Get(i int) bool {
	if i < 0 || i >= pv.size {
		return false
	}
	return pv.Rank1(i+1)-pv.Rank1(i) == 1
}

// Build constructs the compressed block representation and rank prefixes.
func (pv *PoppyVector) Build() {
	if pv.size == 0 {
		pv.entries = nil
		pv.superPrefix = []int{0}
		pv.blockPrefix = nil
		pv.data = nil
		return
	}

	nBlocks := (pv.size + poppyBlockSize - 1) / poppyBlockSize
	nSupers := (nBlocks + poppyBlocksPerSuper - 1) / poppyBlocksPerSuper

	pv.entries = make([]uint32, nBlocks)
	pv.blockPrefix = make([]uint16, nBlocks)
	pv.superPrefix = make([]int, nSupers+1)

	totalOnes := 0
	for super := 0; super < nSupers; super++ {
		pv.superPrefix[super] = totalOnes
		withinSuper := 0

		blockStart := super * poppyBlocksPerSuper
		blockEnd := blockStart + poppyBlocksPerSuper
		if blockEnd > nBlocks {
			blockEnd = nBlocks
		}

		for block := blockStart; block < blockEnd; block++ {
			pv.blockPrefix[block] = uint16(withinSuper)

			start := block * poppyBlockSize
			blockLen := poppyBlockSize
			if remain := pv.size - start; remain < blockLen {
				blockLen = remain
			}

			bits := uint16(0)
			for j := 0; j < blockLen; j++ {
				idx := start + j
				if (pv.data[idx/64]>>uint(idx%64))&1 == 1 {
					bits |= 1 << uint(j)
				}
			}

			class, offset := rrrEncode(bits, blockLen)
			pv.entries[block] = uint32(class) | (uint32(offset) << 8)
			withinSuper += class
		}

		totalOnes += withinSuper
	}
	pv.superPrefix[nSupers] = totalOnes

	// Raw words are no longer needed after the compact representation is built.
	pv.data = nil
}

// Rank1 returns the number of 1 bits in [0, i).
func (pv *PoppyVector) Rank1(i int) int {
	if i <= 0 {
		return 0
	}
	if i > pv.size {
		i = pv.size
	}
	if pv.size == 0 {
		return 0
	}

	block := i / poppyBlockSize
	bit := i % poppyBlockSize
	if block >= len(pv.entries) {
		return pv.superPrefix[len(pv.superPrefix)-1]
	}

	super := block / poppyBlocksPerSuper
	count := pv.superPrefix[super] + int(pv.blockPrefix[block])

	if bit > 0 {
		entry := pv.entries[block]
		class := int(entry & 0xff)
		offset := int(entry >> 8)

		start := block * poppyBlockSize
		blockLen := poppyBlockSize
		if remain := pv.size - start; remain < blockLen {
			blockLen = remain
		}

		count += rrrRankPrefix(class, offset, blockLen, bit)
	}
	return count
}

// Rank0 returns the number of 0 bits in [0, i).
func (pv *PoppyVector) Rank0(i int) int {
	if i <= 0 {
		return 0
	}
	if i > pv.size {
		i = pv.size
	}
	return i - pv.Rank1(i)
}

// Size returns the number of bits in the vector.
func (pv *PoppyVector) Size() int { return pv.size }

// TotalOnes returns the total number of 1 bits.
func (pv *PoppyVector) TotalOnes() int {
	if len(pv.superPrefix) == 0 {
		return 0
	}
	return pv.superPrefix[len(pv.superPrefix)-1]
}
