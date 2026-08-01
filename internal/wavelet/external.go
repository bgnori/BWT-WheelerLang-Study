package wavelet

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"os"
)

const (
	externalSuperBlockBits  = 4096
	externalSuperBlockWords = externalSuperBlockBits / 64
	defaultDiskBlockSize    = 4096
)

// ExternalConfig controls physical external-memory behavior.
type ExternalConfig struct {
	DiskBlockSize int
}

// ExternalTree is a Wavelet Tree variant that stores node bit-vectors in
// temporary files (external memory) and keeps only rank summaries in memory.
type ExternalTree struct {
	n    int
	root *externalNode
}

type externalNode struct {
	f        *os.File
	n        int
	words    int
	blockSz  int
	super    []int
	totalOne int
	curWord  uint64
	curIdx   int
	hasCur   bool
	cacheOff int64
	cache    []byte
	left     *externalNode
	right    *externalNode
}

// BuildExternal constructs an external-memory Wavelet Tree from seq.
func BuildExternal(seq []byte) *ExternalTree {
	return BuildExternalWithConfig(seq, ExternalConfig{})
}

// BuildExternalWithConfig constructs an external-memory Wavelet Tree from seq
// with explicit physical storage settings.
func BuildExternalWithConfig(seq []byte, cfg ExternalConfig) *ExternalTree {
	t := &ExternalTree{n: len(seq)}
	if len(seq) > 0 {
		t.root = buildExternalNode(seq, 0, 256, normalizeExternalConfig(cfg))
	}
	return t
}

func buildExternalNode(seq []byte, lo, hi int, cfg ExternalConfig) *externalNode {
	if len(seq) == 0 || hi-lo <= 1 {
		return nil
	}
	mid := (lo + hi) / 2
	nd := newExternalNode(len(seq), cfg.DiskBlockSize)

	leftSeq := make([]byte, 0, len(seq))
	rightSeq := make([]byte, 0, len(seq))
	for i, c := range seq {
		if int(c) >= mid {
			nd.setBit(i)
			rightSeq = append(rightSeq, c)
		} else {
			leftSeq = append(leftSeq, c)
		}
	}
	nd.finalize()
	nd.left = buildExternalNode(leftSeq, lo, mid, cfg)
	nd.right = buildExternalNode(rightSeq, mid, hi, cfg)
	return nd
}

// Rank returns the number of occurrences of c in seq[0..i).
func (t *ExternalTree) Rank(c byte, i int) int {
	if t.root == nil || i <= 0 {
		return 0
	}
	if i > t.n {
		i = t.n
	}
	return rankExternalNode(t.root, int(c), i, 0, 256)
}

func rankExternalNode(nd *externalNode, c, i, lo, hi int) int {
	if i <= 0 {
		return 0
	}
	if hi-lo <= 1 {
		if c == lo {
			return i
		}
		return 0
	}
	if nd == nil {
		return 0
	}
	mid := (lo + hi) / 2
	ones := nd.rank1(i)
	if c < mid {
		return rankExternalNode(nd.left, c, i-ones, lo, mid)
	}
	return rankExternalNode(nd.right, c, ones, mid, hi)
}

func normalizeExternalConfig(cfg ExternalConfig) ExternalConfig {
	if cfg.DiskBlockSize <= 0 {
		cfg.DiskBlockSize = defaultDiskBlockSize
	}
	if cfg.DiskBlockSize < 8 {
		cfg.DiskBlockSize = 8
	}
	if rem := cfg.DiskBlockSize % 8; rem != 0 {
		cfg.DiskBlockSize += 8 - rem
	}
	return cfg
}

func newExternalNode(n int, blockSize int) *externalNode {
	tmp, err := os.CreateTemp("", "wavelet-ext-*")
	if err != nil {
		panic(fmt.Sprintf("wavelet external: create temp file: %v", err))
	}
	_ = os.Remove(tmp.Name())

	words := (n + 63) / 64
	if err := tmp.Truncate(int64(words * 8)); err != nil {
		panic(fmt.Sprintf("wavelet external: truncate temp file: %v", err))
	}
	numSuper := (n + externalSuperBlockBits - 1) / externalSuperBlockBits
	super := make([]int, numSuper+1)
	return &externalNode{
		f:        tmp,
		n:        n,
		words:    words,
		blockSz:  blockSize,
		super:    super,
		cacheOff: -1,
	}
}

func (nd *externalNode) setBit(i int) {
	wordIdx := i / 64
	bit := uint(i % 64)
	if !nd.hasCur {
		nd.curIdx = wordIdx
		nd.curWord = 0
		nd.hasCur = true
	}
	if wordIdx != nd.curIdx {
		nd.writeWord(nd.curIdx, nd.curWord)
		nd.curIdx = wordIdx
		nd.curWord = 0
	}
	nd.curWord |= 1 << bit
}

func (nd *externalNode) finalize() {
	nd.flushCurrentWord()

	oneCount := 0
	for wordIdx := 0; wordIdx < nd.words; wordIdx++ {
		if wordIdx%externalSuperBlockWords == 0 {
			sb := wordIdx / externalSuperBlockWords
			nd.super[sb] = oneCount
		}
		w := nd.readWord(wordIdx)
		oneCount += bits.OnesCount64(w)
	}
	nd.totalOne = oneCount
	if len(nd.super) > 0 {
		nd.super[len(nd.super)-1] = oneCount
	}
}

func (nd *externalNode) flushCurrentWord() {
	if !nd.hasCur {
		return
	}
	nd.writeWord(nd.curIdx, nd.curWord)
	nd.hasCur = false
	nd.curWord = 0
}

func (nd *externalNode) writeWord(wordIdx int, word uint64) {
	if wordIdx < 0 || wordIdx >= nd.words {
		return
	}
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], word)
	off := int64(wordIdx * 8)
	if _, err := nd.f.WriteAt(buf[:], off); err != nil {
		panic(fmt.Sprintf("wavelet external: write temp word: %v", err))
	}
	if nd.cacheOff >= 0 {
		nd.cacheOff = -1
		nd.cache = nd.cache[:0]
	}
}

func (nd *externalNode) rank1(i int) int {
	if i <= 0 {
		return 0
	}
	if i > nd.n {
		i = nd.n
	}
	if i == nd.n {
		return nd.totalOne
	}

	sb := i / externalSuperBlockBits
	if sb >= len(nd.super)-1 {
		sb = len(nd.super) - 1
	}
	count := nd.super[sb]

	startWord := sb * externalSuperBlockWords
	endWord := i / 64
	for w := startWord; w < endWord; w++ {
		count += bits.OnesCount64(nd.readWord(w))
	}

	if rem := uint(i % 64); rem > 0 {
		word := nd.readWord(endWord)
		count += bits.OnesCount64(word & ((uint64(1) << rem) - 1))
	}
	return count
}

func (nd *externalNode) readWord(wordIdx int) uint64 {
	if wordIdx < 0 || wordIdx >= nd.words {
		return 0
	}
	off := int64(wordIdx * 8)
	blockOff := (off / int64(nd.blockSz)) * int64(nd.blockSz)
	if nd.cacheOff != blockOff {
		if cap(nd.cache) < nd.blockSz {
			nd.cache = make([]byte, nd.blockSz)
		} else {
			nd.cache = nd.cache[:nd.blockSz]
		}
		n, err := nd.f.ReadAt(nd.cache, blockOff)
		if err != nil && err != io.EOF {
			panic(fmt.Sprintf("wavelet external: read temp block: %v", err))
		}
		for i := n; i < len(nd.cache); i++ {
			nd.cache[i] = 0
		}
		nd.cacheOff = blockOff
	}
	start := int(off - blockOff)
	return binary.LittleEndian.Uint64(nd.cache[start : start+8])
}
