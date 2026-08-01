package fmindex

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"os"
	"sort"
)

const (
	externalBitvecSuperBlockBits  = 4096
	externalBitvecSuperBlockWords = externalBitvecSuperBlockBits / 64
)

type externalBitvecOcc struct {
	vecs [256]*externalBitVector
}

type externalBitVector struct {
	n        int
	words    int
	super    []int
	totalOne int
	curWord  uint64
	curIdx   int
	hasCur   bool
	store    externalBitvecWordStore
}

type externalBitvecWordStore interface {
	WriteWord(wordIdx int, word uint64)
	Finalize()
	ReadWord(wordIdx int) uint64
}

func buildExternalBitvecOcc(bwt []byte, storage OccStorageOptions) *externalBitvecOcc {
	n := len(bwt)
	occ := &externalBitvecOcc{}
	for i, b := range bwt {
		if occ.vecs[b] == nil {
			occ.vecs[b] = newExternalBitVector(n, storage)
		}
		occ.vecs[b].setBit(i)
	}
	for i := range occ.vecs {
		if occ.vecs[i] != nil {
			occ.vecs[i].finalize()
		}
	}
	return occ
}

func (o *externalBitvecOcc) rank(b byte, i int) int {
	if o.vecs[b] == nil {
		return 0
	}
	return o.vecs[b].rank1(i)
}

func newExternalBitVector(n int, storage OccStorageOptions) *externalBitVector {
	words := (n + 63) / 64
	numSuper := (n + externalBitvecSuperBlockBits - 1) / externalBitvecSuperBlockBits
	super := make([]int, numSuper+1)

	var store externalBitvecWordStore
	switch storage.ExternalStrategy {
	case OccExternalStrategyBPlusTree:
		store = newBPlusBitvecWordStore(words, storage.DiskBlockSize)
	case OccExternalStrategyInvertedSegments:
		store = newInvertedBitvecWordStore(words, storage.DiskBlockSize)
	default:
		store = newLSMBitvecWordStore(words, storage.DiskBlockSize)
	}
	return &externalBitVector{n: n, words: words, super: super, store: store}
}

func (v *externalBitVector) setBit(i int) {
	wordIdx := i / 64
	bit := uint(i % 64)
	if !v.hasCur {
		v.curIdx = wordIdx
		v.curWord = 0
		v.hasCur = true
	}
	if wordIdx != v.curIdx {
		v.store.WriteWord(v.curIdx, v.curWord)
		v.curIdx = wordIdx
		v.curWord = 0
	}
	v.curWord |= 1 << bit
}

func (v *externalBitVector) finalize() {
	if v.hasCur {
		v.store.WriteWord(v.curIdx, v.curWord)
		v.hasCur = false
		v.curWord = 0
	}
	v.store.Finalize()

	oneCount := 0
	for wordIdx := 0; wordIdx < v.words; wordIdx++ {
		if wordIdx%externalBitvecSuperBlockWords == 0 {
			sb := wordIdx / externalBitvecSuperBlockWords
			v.super[sb] = oneCount
		}
		oneCount += bits.OnesCount64(v.store.ReadWord(wordIdx))
	}
	v.totalOne = oneCount
	if len(v.super) > 0 {
		v.super[len(v.super)-1] = oneCount
	}
}

func (v *externalBitVector) rank1(i int) int {
	if i <= 0 {
		return 0
	}
	if i > v.n {
		i = v.n
	}
	if i == v.n {
		return v.totalOne
	}

	sb := i / externalBitvecSuperBlockBits
	if sb >= len(v.super)-1 {
		sb = len(v.super) - 1
	}
	count := v.super[sb]

	startWord := sb * externalBitvecSuperBlockWords
	endWord := i / 64
	for w := startWord; w < endWord; w++ {
		count += bits.OnesCount64(v.store.ReadWord(w))
	}
	if rem := uint(i % 64); rem > 0 {
		word := v.store.ReadWord(endWord)
		count += bits.OnesCount64(word & ((uint64(1) << rem) - 1))
	}
	return count
}

type denseBitvecWordReader struct {
	f        *os.File
	words    int
	blockSz  int
	cacheOff int64
	cache    []byte
}

func newDenseBitvecWordReader(words, blockSize int) *denseBitvecWordReader {
	tmp, err := os.CreateTemp("", "fmidx-ext-bitvec-*")
	if err != nil {
		panic(fmt.Sprintf("fmindex external bitvec: create temp file: %v", err))
	}
	_ = os.Remove(tmp.Name())
	if err := tmp.Truncate(int64(words * 8)); err != nil {
		panic(fmt.Sprintf("fmindex external bitvec: truncate temp file: %v", err))
	}
	return &denseBitvecWordReader{f: tmp, words: words, blockSz: blockSize, cacheOff: -1}
}

func (d *denseBitvecWordReader) writeWordAt(wordIdx int, word uint64) {
	if wordIdx < 0 || wordIdx >= d.words {
		return
	}
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], word)
	off := int64(wordIdx * 8)
	if _, err := d.f.WriteAt(buf[:], off); err != nil {
		panic(fmt.Sprintf("fmindex external bitvec: write temp word: %v", err))
	}
	if d.cacheOff >= 0 {
		d.cacheOff = -1
		d.cache = d.cache[:0]
	}
}

func (d *denseBitvecWordReader) readWordAt(wordIdx int) uint64 {
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
			panic(fmt.Sprintf("fmindex external bitvec: read temp block: %v", err))
		}
		for i := n; i < len(d.cache); i++ {
			d.cache[i] = 0
		}
		d.cacheOff = blockOff
	}
	start := int(off - blockOff)
	return binary.LittleEndian.Uint64(d.cache[start : start+8])
}

type lsmBitvecWordStore struct {
	wal   *os.File
	dense *denseBitvecWordReader
	words int
}

func newLSMBitvecWordStore(words, blockSize int) *lsmBitvecWordStore {
	wal, err := os.CreateTemp("", "fmidx-ext-bitvec-lsm-wal-*")
	if err != nil {
		panic(fmt.Sprintf("fmindex external bitvec: create lsm wal: %v", err))
	}
	_ = os.Remove(wal.Name())
	return &lsmBitvecWordStore{wal: wal, dense: newDenseBitvecWordReader(words, blockSize), words: words}
}

func (s *lsmBitvecWordStore) WriteWord(wordIdx int, word uint64) {
	if wordIdx < 0 || wordIdx >= s.words {
		return
	}
	var buf [12]byte
	binary.LittleEndian.PutUint32(buf[0:4], uint32(wordIdx))
	binary.LittleEndian.PutUint64(buf[4:12], word)
	if _, err := s.wal.Write(buf[:]); err != nil {
		panic(fmt.Sprintf("fmindex external bitvec: append lsm wal: %v", err))
	}
}

func (s *lsmBitvecWordStore) Finalize() {
	if _, err := s.wal.Seek(0, io.SeekStart); err != nil {
		panic(fmt.Sprintf("fmindex external bitvec: seek lsm wal: %v", err))
	}
	var rec [12]byte
	for {
		_, err := io.ReadFull(s.wal, rec[:])
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			panic(fmt.Sprintf("fmindex external bitvec: read lsm wal: %v", err))
		}
		wordIdx := int(binary.LittleEndian.Uint32(rec[0:4]))
		word := binary.LittleEndian.Uint64(rec[4:12])
		s.dense.writeWordAt(wordIdx, word)
	}
}

func (s *lsmBitvecWordStore) ReadWord(wordIdx int) uint64 {
	return s.dense.readWordAt(wordIdx)
}

type bPlusBitvecWordStore struct {
	dense      *denseBitvecWordReader
	words      int
	pageStarts []int
}

func newBPlusBitvecWordStore(words, blockSize int) *bPlusBitvecWordStore {
	const pageWords = 512
	pages := (words + pageWords - 1) / pageWords
	starts := make([]int, pages)
	for i := range starts {
		starts[i] = i * pageWords
	}
	return &bPlusBitvecWordStore{dense: newDenseBitvecWordReader(words, blockSize), words: words, pageStarts: starts}
}

func (s *bPlusBitvecWordStore) WriteWord(wordIdx int, word uint64) {
	s.dense.writeWordAt(wordIdx, word)
}

func (s *bPlusBitvecWordStore) Finalize() {}

func (s *bPlusBitvecWordStore) ReadWord(wordIdx int) uint64 {
	if wordIdx < 0 || wordIdx >= s.words {
		return 0
	}
	page := sort.Search(len(s.pageStarts), func(i int) bool { return s.pageStarts[i] > wordIdx }) - 1
	if page < 0 {
		page = 0
	}
	_ = page
	return s.dense.readWordAt(wordIdx)
}

type invBitvecEntry struct {
	wordIdx int
	word    uint64
}

type invBitvecSegment struct {
	start int
	off   int64
}

type invBitvecShard struct {
	f        *os.File
	entries  int
	segments []invBitvecSegment
}

type invertedBitvecWordStore struct {
	words      int
	shards     []invBitvecShard
	shardCount int
}

func newInvertedBitvecWordStore(words, _blockSize int) *invertedBitvecWordStore {
	const shardCount = 8
	store := &invertedBitvecWordStore{words: words, shards: make([]invBitvecShard, shardCount), shardCount: shardCount}
	for i := range store.shards {
		tmp, err := os.CreateTemp("", "fmidx-ext-bitvec-inv-*")
		if err != nil {
			panic(fmt.Sprintf("fmindex external bitvec: create inverted shard: %v", err))
		}
		_ = os.Remove(tmp.Name())
		store.shards[i].f = tmp
	}
	return store
}

func (s *invertedBitvecWordStore) WriteWord(wordIdx int, word uint64) {
	if wordIdx < 0 || wordIdx >= s.words {
		return
	}
	sh := wordIdx % s.shardCount
	shard := &s.shards[sh]
	var buf [12]byte
	binary.LittleEndian.PutUint32(buf[0:4], uint32(wordIdx))
	binary.LittleEndian.PutUint64(buf[4:12], word)
	if _, err := shard.f.Write(buf[:]); err != nil {
		panic(fmt.Sprintf("fmindex external bitvec: append inverted entry: %v", err))
	}
	shard.entries++
}

func (s *invertedBitvecWordStore) Finalize() {
	const segmentWords = 256
	for i := range s.shards {
		shard := &s.shards[i]
		if _, err := shard.f.Seek(0, io.SeekStart); err != nil {
			panic(fmt.Sprintf("fmindex external bitvec: seek inverted shard: %v", err))
		}
		entries := make([]invBitvecEntry, shard.entries)
		var rec [12]byte
		for j := 0; j < shard.entries; j++ {
			if _, err := io.ReadFull(shard.f, rec[:]); err != nil {
				panic(fmt.Sprintf("fmindex external bitvec: read inverted entry: %v", err))
			}
			entries[j] = invBitvecEntry{wordIdx: int(binary.LittleEndian.Uint32(rec[0:4])), word: binary.LittleEndian.Uint64(rec[4:12])}
		}
		sort.Slice(entries, func(a, b int) bool { return entries[a].wordIdx < entries[b].wordIdx })
		if err := shard.f.Truncate(0); err != nil {
			panic(fmt.Sprintf("fmindex external bitvec: truncate inverted shard: %v", err))
		}
		if _, err := shard.f.Seek(0, io.SeekStart); err != nil {
			panic(fmt.Sprintf("fmindex external bitvec: rewind inverted shard: %v", err))
		}
		shard.segments = shard.segments[:0]
		for j, e := range entries {
			if j%segmentWords == 0 {
				shard.segments = append(shard.segments, invBitvecSegment{start: e.wordIdx, off: int64(j * 12)})
			}
			var out [12]byte
			binary.LittleEndian.PutUint32(out[0:4], uint32(e.wordIdx))
			binary.LittleEndian.PutUint64(out[4:12], e.word)
			if _, err := shard.f.Write(out[:]); err != nil {
				panic(fmt.Sprintf("fmindex external bitvec: rewrite inverted shard: %v", err))
			}
		}
	}
}

func (s *invertedBitvecWordStore) ReadWord(wordIdx int) uint64 {
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
	startEntry := int(shard.segments[seg].off / 12)
	endEntry := shard.entries
	if seg+1 < len(shard.segments) {
		endEntry = int(shard.segments[seg+1].off / 12)
	}
	var rec [12]byte
	for i := startEntry; i < endEntry; i++ {
		off := int64(i * 12)
		if _, err := shard.f.ReadAt(rec[:], off); err != nil {
			panic(fmt.Sprintf("fmindex external bitvec: read inverted shard at: %v", err))
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
