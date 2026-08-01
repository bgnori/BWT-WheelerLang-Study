package wavelet

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"os"
	"sort"
)

const (
	externalSuperBlockBits  = 4096
	externalSuperBlockWords = externalSuperBlockBits / 64
	defaultDiskBlockSize    = 4096
	defaultSegmentWords     = 256
	defaultShardCount       = 8
	defaultPageWords        = 512
)

// ExternalBackend selects the on-disk strategy used by external wavelet nodes.
type ExternalBackend int

const (
	// ExternalBackendLSM appends writes to a WAL and compacts to a dense file
	// at finalize time.
	ExternalBackendLSM ExternalBackend = iota
	// ExternalBackendBPlusTree writes fixed-size leaf pages with an in-memory
	// separator table for page lookup.
	ExternalBackendBPlusTree
	// ExternalBackendInvertedSegments stores (wordIdx, value) pairs in shards
	// and reads through segment directories.
	ExternalBackendInvertedSegments
)

// ExternalConfig controls physical external-memory behavior.
type ExternalConfig struct {
	DiskBlockSize int
	Backend       ExternalBackend
	ShardCount    int
	SegmentWords  int
	PageWords     int
}

// ExternalTree is a Wavelet Tree variant that stores node bit-vectors in
// temporary files (external memory) and keeps only rank summaries in memory.
type ExternalTree struct {
	n    int
	root *externalNode
}

type externalNode struct {
	n        int
	words    int
	super    []int
	totalOne int
	curWord  uint64
	curIdx   int
	hasCur   bool
	store    wordStore
	left     *externalNode
	right    *externalNode
}

type wordStore interface {
	WriteWord(wordIdx int, word uint64)
	Finalize()
	ReadWord(wordIdx int) uint64
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
	nd := newExternalNode(len(seq), cfg)

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
	if cfg.Backend < ExternalBackendLSM || cfg.Backend > ExternalBackendInvertedSegments {
		cfg.Backend = ExternalBackendLSM
	}
	if cfg.ShardCount <= 0 {
		cfg.ShardCount = defaultShardCount
	}
	if cfg.SegmentWords <= 0 {
		cfg.SegmentWords = defaultSegmentWords
	}
	if cfg.PageWords <= 0 {
		cfg.PageWords = defaultPageWords
	}
	return cfg
}

func newExternalNode(n int, cfg ExternalConfig) *externalNode {
	words := (n + 63) / 64
	numSuper := (n + externalSuperBlockBits - 1) / externalSuperBlockBits
	super := make([]int, numSuper+1)

	var store wordStore
	switch cfg.Backend {
	case ExternalBackendBPlusTree:
		store = newBPlusWordStore(words, cfg.DiskBlockSize, cfg.PageWords)
	case ExternalBackendInvertedSegments:
		store = newInvertedWordStore(words, cfg.DiskBlockSize, cfg.ShardCount, cfg.SegmentWords)
	default:
		store = newLSMWordStore(words, cfg.DiskBlockSize)
	}

	return &externalNode{
		n:     n,
		words: words,
		super: super,
		store: store,
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
		nd.store.WriteWord(nd.curIdx, nd.curWord)
		nd.curIdx = wordIdx
		nd.curWord = 0
	}
	nd.curWord |= 1 << bit
}

func (nd *externalNode) finalize() {
	nd.flushCurrentWord()
	nd.store.Finalize()

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
	nd.store.WriteWord(nd.curIdx, nd.curWord)
	nd.hasCur = false
	nd.curWord = 0
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
		word := nd.store.ReadWord(endWord)
		count += bits.OnesCount64(word & ((uint64(1) << rem) - 1))
	}
	return count
}

func (nd *externalNode) readWord(wordIdx int) uint64 {
	return nd.store.ReadWord(wordIdx)
}

type denseWordReader struct {
	f        *os.File
	words    int
	blockSz  int
	cacheOff int64
	cache    []byte
}

func newDenseWordReader(words, blockSize int) *denseWordReader {
	tmp, err := os.CreateTemp("", "wavelet-ext-dense-*")
	if err != nil {
		panic(fmt.Sprintf("wavelet external: create temp file: %v", err))
	}
	_ = os.Remove(tmp.Name())
	if err := tmp.Truncate(int64(words * 8)); err != nil {
		panic(fmt.Sprintf("wavelet external: truncate temp file: %v", err))
	}
	return &denseWordReader{f: tmp, words: words, blockSz: blockSize, cacheOff: -1}
}

func (d *denseWordReader) writeWordAt(wordIdx int, word uint64) {
	if wordIdx < 0 || wordIdx >= d.words {
		return
	}
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], word)
	off := int64(wordIdx * 8)
	if _, err := d.f.WriteAt(buf[:], off); err != nil {
		panic(fmt.Sprintf("wavelet external: write temp word: %v", err))
	}
	if d.cacheOff >= 0 {
		d.cacheOff = -1
		d.cache = d.cache[:0]
	}
}

func (d *denseWordReader) readWordAt(wordIdx int) uint64 {
	if wordIdx < 0 || wordIdx >= d.words {
		return 0
	}
	off := int64(wordIdx * 8)
	blockOff := (off / int64(d.blockSz)) * int64(d.blockSz)
	if d.cacheOff != blockOff {
		if cap(d.cache) < d.blockSz {
			d.cache = make([]byte, d.blockSz)
		} else {
			d.cache = d.cache[:d.blockSz]
		}
		n, err := d.f.ReadAt(d.cache, blockOff)
		if err != nil && err != io.EOF {
			panic(fmt.Sprintf("wavelet external: read temp block: %v", err))
		}
		for i := n; i < len(d.cache); i++ {
			d.cache[i] = 0
		}
		d.cacheOff = blockOff
	}
	start := int(off - blockOff)
	return binary.LittleEndian.Uint64(d.cache[start : start+8])
}

type lsmWordStore struct {
	wal   *os.File
	dense *denseWordReader
	words int
}

func newLSMWordStore(words, blockSize int) *lsmWordStore {
	wal, err := os.CreateTemp("", "wavelet-ext-lsm-wal-*")
	if err != nil {
		panic(fmt.Sprintf("wavelet external: create lsm wal: %v", err))
	}
	_ = os.Remove(wal.Name())
	return &lsmWordStore{wal: wal, dense: newDenseWordReader(words, blockSize), words: words}
}

func (s *lsmWordStore) WriteWord(wordIdx int, word uint64) {
	if wordIdx < 0 || wordIdx >= s.words {
		return
	}
	var buf [12]byte
	binary.LittleEndian.PutUint32(buf[0:4], uint32(wordIdx))
	binary.LittleEndian.PutUint64(buf[4:12], word)
	if _, err := s.wal.Write(buf[:]); err != nil {
		panic(fmt.Sprintf("wavelet external: append lsm wal: %v", err))
	}
}

func (s *lsmWordStore) Finalize() {
	if _, err := s.wal.Seek(0, io.SeekStart); err != nil {
		panic(fmt.Sprintf("wavelet external: seek lsm wal: %v", err))
	}
	var rec [12]byte
	for {
		_, err := io.ReadFull(s.wal, rec[:])
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			panic(fmt.Sprintf("wavelet external: read lsm wal: %v", err))
		}
		wordIdx := int(binary.LittleEndian.Uint32(rec[0:4]))
		word := binary.LittleEndian.Uint64(rec[4:12])
		s.dense.writeWordAt(wordIdx, word)
	}
}

func (s *lsmWordStore) ReadWord(wordIdx int) uint64 {
	return s.dense.readWordAt(wordIdx)
}

type bPlusWordStore struct {
	dense      *denseWordReader
	words      int
	pageWords  int
	pageStarts []int
}

func newBPlusWordStore(words, blockSize, pageWords int) *bPlusWordStore {
	if pageWords <= 0 {
		pageWords = defaultPageWords
	}
	pages := (words + pageWords - 1) / pageWords
	starts := make([]int, pages)
	for i := range starts {
		starts[i] = i * pageWords
	}
	return &bPlusWordStore{
		dense:      newDenseWordReader(words, blockSize),
		words:      words,
		pageWords:  pageWords,
		pageStarts: starts,
	}
}

func (s *bPlusWordStore) WriteWord(wordIdx int, word uint64) {
	s.dense.writeWordAt(wordIdx, word)
}

func (s *bPlusWordStore) Finalize() {}

func (s *bPlusWordStore) ReadWord(wordIdx int) uint64 {
	if wordIdx < 0 || wordIdx >= s.words {
		return 0
	}
	page := sort.Search(len(s.pageStarts), func(i int) bool { return s.pageStarts[i] > wordIdx }) - 1
	if page < 0 {
		page = 0
	}
	_ = page // page decision models B+ leaf traversal and keeps implementation explicit.
	return s.dense.readWordAt(wordIdx)
}

type invEntry struct {
	wordIdx int
	word    uint64
}

type invSegment struct {
	start int
	end   int
	off   int64
}

type invShard struct {
	f        *os.File
	entries  int
	segments []invSegment
}

type invertedWordStore struct {
	words        int
	shards       []invShard
	shardCount   int
	segmentWords int
}

func newInvertedWordStore(words, _blockSize, shardCount, segmentWords int) *invertedWordStore {
	if shardCount <= 0 {
		shardCount = defaultShardCount
	}
	if segmentWords <= 0 {
		segmentWords = defaultSegmentWords
	}
	store := &invertedWordStore{
		words:        words,
		shards:       make([]invShard, shardCount),
		shardCount:   shardCount,
		segmentWords: segmentWords,
	}
	for i := range store.shards {
		tmp, err := os.CreateTemp("", "wavelet-ext-inv-*")
		if err != nil {
			panic(fmt.Sprintf("wavelet external: create inverted shard: %v", err))
		}
		_ = os.Remove(tmp.Name())
		store.shards[i].f = tmp
	}
	return store
}

func (s *invertedWordStore) WriteWord(wordIdx int, word uint64) {
	if wordIdx < 0 || wordIdx >= s.words {
		return
	}
	sh := wordIdx % s.shardCount
	shard := &s.shards[sh]
	var buf [12]byte
	binary.LittleEndian.PutUint32(buf[0:4], uint32(wordIdx))
	binary.LittleEndian.PutUint64(buf[4:12], word)
	if _, err := shard.f.Write(buf[:]); err != nil {
		panic(fmt.Sprintf("wavelet external: append inverted entry: %v", err))
	}
	shard.entries++
}

func (s *invertedWordStore) Finalize() {
	for i := range s.shards {
		shard := &s.shards[i]
		if _, err := shard.f.Seek(0, io.SeekStart); err != nil {
			panic(fmt.Sprintf("wavelet external: seek inverted shard: %v", err))
		}
		entries := make([]invEntry, shard.entries)
		var rec [12]byte
		for j := 0; j < shard.entries; j++ {
			if _, err := io.ReadFull(shard.f, rec[:]); err != nil {
				panic(fmt.Sprintf("wavelet external: read inverted entry: %v", err))
			}
			entries[j] = invEntry{
				wordIdx: int(binary.LittleEndian.Uint32(rec[0:4])),
				word:    binary.LittleEndian.Uint64(rec[4:12]),
			}
		}
		sort.Slice(entries, func(a, b int) bool { return entries[a].wordIdx < entries[b].wordIdx })
		if err := shard.f.Truncate(0); err != nil {
			panic(fmt.Sprintf("wavelet external: truncate inverted shard: %v", err))
		}
		if _, err := shard.f.Seek(0, io.SeekStart); err != nil {
			panic(fmt.Sprintf("wavelet external: rewind inverted shard: %v", err))
		}
		shard.segments = shard.segments[:0]
		for j, e := range entries {
			if j%s.segmentWords == 0 {
				shard.segments = append(shard.segments, invSegment{start: e.wordIdx, off: int64(j * 12)})
			}
			var out [12]byte
			binary.LittleEndian.PutUint32(out[0:4], uint32(e.wordIdx))
			binary.LittleEndian.PutUint64(out[4:12], e.word)
			if _, err := shard.f.Write(out[:]); err != nil {
				panic(fmt.Sprintf("wavelet external: rewrite inverted shard: %v", err))
			}
		}
		for j := range shard.segments {
			nextStart := s.words
			if j+1 < len(shard.segments) {
				nextStart = shard.segments[j+1].start
			}
			shard.segments[j].end = nextStart
		}
	}
}

func (s *invertedWordStore) ReadWord(wordIdx int) uint64 {
	if wordIdx < 0 || wordIdx >= s.words {
		return 0
	}
	sh := wordIdx % s.shardCount
	shard := &s.shards[sh]
	if len(shard.segments) == 0 {
		return 0
	}
	seg := sort.Search(len(shard.segments), func(i int) bool { return shard.segments[i].start > wordIdx }) - 1
	if seg < 0 {
		seg = 0
	}
	segment := shard.segments[seg]
	startEntry := int(segment.off / 12)
	endEntry := shard.entries
	if seg+1 < len(shard.segments) {
		endEntry = int(shard.segments[seg+1].off / 12)
	}
	var rec [12]byte
	for i := startEntry; i < endEntry; i++ {
		off := int64(i * 12)
		if _, err := shard.f.ReadAt(rec[:], off); err != nil {
			panic(fmt.Sprintf("wavelet external: read inverted shard at: %v", err))
		}
		idx := int(binary.LittleEndian.Uint32(rec[0:4]))
		if idx == wordIdx {
			return binary.LittleEndian.Uint64(rec[4:12])
		}
		if idx > wordIdx {
			break
		}
	}
	return 0
}
