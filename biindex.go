package bwtsearch

// Package bwtsearch – bidirectional FM-index (BiIndex).
//
// A Bidirectional FM-index stores two FM-indexes simultaneously:
//
//   - fwd: an FM-index of the original text T
//   - rev: an FM-index of the reverse text T^R
//
// Together they support both left-extension (standard backward search) and
// right-extension of a pattern interval without rebuilding either index.
// This enables efficient bidirectional pattern matching, approximate search,
// and seed-and-extend alignment strategies.
//
// The bidirectional backward-search algorithm (Lam et al. 2009 / Schnattinger
// et al. 2010) maintains a paired interval (BiInterval) such that:
//
//   - [LoFwd, HiFwd) is the SA range in fwd for the current pattern P
//   - [LoRev, HiRev) is the SA range in rev for P reversed (P^R)
//
// These two ranges always represent exactly the same set of occurrences.
// Extending P to the left by c updates both ranges using fwd's rank structure;
// extending P to the right by c updates both ranges using rev's rank structure.

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/bgnori/textindex/internal/fmindex"
)

const biMagic = "BIDX001"

// BiInterval is a paired suffix-array interval used during bidirectional search.
// [LoFwd, HiFwd) is the interval in the forward FM-index (of T).
// [LoRev, HiRev) is the interval in the reverse FM-index (of T^R).
type BiInterval struct {
	LoFwd, HiFwd int
	LoRev, HiRev int
}

// Size returns the number of occurrences represented by this interval.
func (bi BiInterval) Size() int {
	s := bi.HiFwd - bi.LoFwd
	if s < 0 {
		return 0
	}
	return s
}

// BiIndex is a bidirectional FM-index supporting both left- and right-extension
// of a search interval.
type BiIndex struct {
	fwd *fmindex.Index // FM-index of T
	rev *fmindex.Index // FM-index of T^R
}

// BuildBi constructs a BiIndex from text using default options.
func BuildBi(text []byte) *BiIndex {
	return BuildBiWithOptions(text, AlgorithmSAIS, OccRLBWT)
}

// BuildBiWithOptions constructs a BiIndex with explicit suffix-array
// construction algorithm and occurrence-array structure.
func BuildBiWithOptions(text []byte, algo SuffixArrayAlgorithm, occ OccStructure) *BiIndex {
	return BuildBiWithConfig(text, algo, occ, OccStorageOptions{Mode: OccStorageInMemory})
}

// BuildBiWithConfig constructs a BiIndex with explicit logical and physical
// occurrence-array options.
func BuildBiWithConfig(text []byte, algo SuffixArrayAlgorithm, occ OccStructure, storage OccStorageOptions) *BiIndex {
	rev := reverseBytes(text)
	innerStorage := fmindex.OccStorageOptions{
		Mode:             fmindex.OccStorageMode(storage.Mode),
		DiskBlockSize:    storage.DiskBlockSize,
		ExternalStrategy: fmindex.OccExternalStrategy(storage.ExternalStrategy),
	}
	return &BiIndex{
		fwd: fmindex.BuildWithConfig(text, fmindex.SuffixArrayAlgorithm(algo), fmindex.OccStructure(occ), innerStorage),
		rev: fmindex.BuildWithConfig(rev, fmindex.SuffixArrayAlgorithm(algo), fmindex.OccStructure(occ), innerStorage),
	}
}

// BuildBiFromFiles concatenates texts with separator and builds a single
// BiIndex over the combined corpus.  If separator is nil a newline (\n) is
// used.  The separator must not contain 0x00.
//
// BuildBiFromFiles panics if separator contains the byte 0x00.
func BuildBiFromFiles(texts [][]byte, separator []byte) *BiIndex {
	return BuildBiFromFilesWithOptions(texts, separator, AlgorithmSAIS, OccRLBWT)
}

// BuildBiFromFilesWithOptions concatenates texts with separator and builds a
// BiIndex using the specified algorithm and occurrence-array structure.
//
// BuildBiFromFilesWithOptions panics if separator contains the byte 0x00.
func BuildBiFromFilesWithOptions(texts [][]byte, separator []byte, algo SuffixArrayAlgorithm, occ OccStructure) *BiIndex {
	return BuildBiFromFilesWithConfig(texts, separator, algo, occ, OccStorageOptions{Mode: OccStorageInMemory})
}

// BuildBiFromFilesWithConfig concatenates texts with separator and builds a
// BiIndex using the specified logical and physical occ options.
func BuildBiFromFilesWithConfig(texts [][]byte, separator []byte, algo SuffixArrayAlgorithm, occ OccStructure, storage OccStorageOptions) *BiIndex {
	if separator == nil {
		separator = []byte{'\n'}
	}
	for _, b := range separator {
		if b == 0x00 {
			panic("bwtsearch: separator must not contain 0x00 (reserved as sentinel)")
		}
	}
	if len(texts) == 0 {
		return BuildBiWithConfig(nil, algo, occ, storage)
	}
	total := 0
	for _, t := range texts {
		total += len(t)
	}
	total += len(separator) * (len(texts) - 1)
	combined := make([]byte, 0, total)
	for i, t := range texts {
		if i > 0 {
			combined = append(combined, separator...)
		}
		combined = append(combined, t...)
	}
	return BuildBiWithConfig(combined, algo, occ, storage)
}

func reverseBytes(b []byte) []byte {
	r := make([]byte, len(b))
	for i, v := range b {
		r[len(b)-1-i] = v
	}
	return r
}

// TextLen returns the length of the indexed text.
// A nil *BiIndex returns 0.
func (idx *BiIndex) TextLen() int {
	if idx == nil || idx.fwd == nil {
		return 0
	}
	return idx.fwd.TextLen()
}

// ContextAround returns a snippet of text centred on pos.
// A nil *BiIndex returns the empty string.
func (idx *BiIndex) ContextAround(pos, patLen, ctxSize int) string {
	if idx == nil || idx.fwd == nil {
		return ""
	}
	return idx.fwd.ContextAround(pos, patLen, ctxSize)
}

// FullInterval returns the initial BiInterval covering the entire SA.
// A nil *BiIndex returns the zero BiInterval.
func (idx *BiIndex) FullInterval() BiInterval {
	if idx == nil || idx.fwd == nil {
		return BiInterval{}
	}
	n := idx.fwd.SALen()
	return BiInterval{LoFwd: 0, HiFwd: n, LoRev: 0, HiRev: n}
}

// ExtendLeft narrows bi by prepending character c (left extension).
// It uses the forward FM-index to update the forward interval and derives
// the reverse interval from the count of characters < c in the BWT range.
// A nil *BiIndex returns the zero (empty) BiInterval.
func (idx *BiIndex) ExtendLeft(bi BiInterval, c byte) BiInterval {
	if idx == nil || idx.fwd == nil {
		return BiInterval{}
	}
	f := idx.fwd
	newLoFwd := f.CValue(c) + f.OccCount(c, bi.LoFwd)
	newHiFwd := f.CValue(c) + f.OccCount(c, bi.HiFwd)

	// Count characters strictly less than c in BWT_fwd[lo_fwd..hi_fwd).
	countLess := 0
	for b := 0; b < int(c); b++ {
		countLess += f.OccCount(byte(b), bi.HiFwd) - f.OccCount(byte(b), bi.LoFwd)
	}
	newLoRev := bi.LoRev + countLess
	newHiRev := newLoRev + (newHiFwd - newLoFwd)
	return BiInterval{LoFwd: newLoFwd, HiFwd: newHiFwd, LoRev: newLoRev, HiRev: newHiRev}
}

// ExtendRight narrows bi by appending character c (right extension).
// It uses the reverse FM-index to update the reverse interval and derives
// the forward interval accordingly.
// A nil *BiIndex returns the zero (empty) BiInterval.
func (idx *BiIndex) ExtendRight(bi BiInterval, c byte) BiInterval {
	if idx == nil || idx.rev == nil {
		return BiInterval{}
	}
	r := idx.rev
	newLoRev := r.CValue(c) + r.OccCount(c, bi.LoRev)
	newHiRev := r.CValue(c) + r.OccCount(c, bi.HiRev)

	// Count characters strictly less than c in BWT_rev[lo_rev..hi_rev).
	countLess := 0
	for b := 0; b < int(c); b++ {
		countLess += r.OccCount(byte(b), bi.HiRev) - r.OccCount(byte(b), bi.LoRev)
	}
	newLoFwd := bi.LoFwd + countLess
	newHiFwd := newLoFwd + (newHiRev - newLoRev)
	return BiInterval{LoFwd: newLoFwd, HiFwd: newHiFwd, LoRev: newLoRev, HiRev: newHiRev}
}

// Count returns the number of occurrences of pattern in the indexed text.
// A nil *BiIndex returns 0.
func (idx *BiIndex) Count(pattern []byte) int {
	if idx == nil || idx.fwd == nil {
		return 0
	}
	return idx.fwd.Count(pattern)
}

// Locate returns up to limit text positions where pattern begins.
// When limit <= 0 all positions are returned.
// A nil *BiIndex returns nil.
func (idx *BiIndex) Locate(pattern []byte, limit int) []int {
	if idx == nil || idx.fwd == nil {
		return nil
	}
	return idx.fwd.Locate(pattern, limit)
}

// WriteTo serialises the BiIndex to w in the BIDX001 format.
// It implements io.WriterTo.
// A nil *BiIndex returns an error.
func (idx *BiIndex) WriteTo(w io.Writer) (int64, error) {
	if idx == nil || idx.fwd == nil || idx.rev == nil {
		return 0, fmt.Errorf("nil biindex")
	}
	var total int64

	n, err := io.WriteString(w, biMagic)
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Serialise the forward index into a temporary buffer so we can prefix
	// its byte length, allowing random-access skipping on load if needed.
	var fwdBuf bytes.Buffer
	if _, werr := idx.fwd.WriteTo(&fwdBuf); werr != nil {
		return total, fmt.Errorf("biindex: serialise fwd: %w", werr)
	}
	if err := binary.Write(w, binary.LittleEndian, int64(fwdBuf.Len())); err != nil {
		return total, err
	}
	total += 8
	n2, err := w.Write(fwdBuf.Bytes())
	total += int64(n2)
	if err != nil {
		return total, err
	}

	// Serialise the reverse index.
	var revBuf bytes.Buffer
	if _, werr := idx.rev.WriteTo(&revBuf); werr != nil {
		return total, fmt.Errorf("biindex: serialise rev: %w", werr)
	}
	if err := binary.Write(w, binary.LittleEndian, int64(revBuf.Len())); err != nil {
		return total, err
	}
	total += 8
	n3, err := w.Write(revBuf.Bytes())
	total += int64(n3)
	return total, err
}

// Save writes the BiIndex to path.
func (idx *BiIndex) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create biindex: %w", err)
	}
	defer f.Close()
	if _, err := idx.WriteTo(f); err != nil {
		return fmt.Errorf("write biindex: %w", err)
	}
	return nil
}

// ReadBiFrom deserialises a BiIndex from r.
// The reader must be positioned at the start of a BIDX001 stream.
func ReadBiFrom(r io.Reader) (*BiIndex, error) {
	br := bufio.NewReaderSize(r, 1<<20)

	hdr := make([]byte, len(biMagic))
	if _, err := io.ReadFull(br, hdr); err != nil {
		return nil, fmt.Errorf("biindex: read magic: %w", err)
	}
	if string(hdr) != biMagic {
		return nil, fmt.Errorf("biindex: bad magic %q (want %q)", string(hdr), biMagic)
	}

	var fwdLen int64
	if err := binary.Read(br, binary.LittleEndian, &fwdLen); err != nil {
		return nil, fmt.Errorf("biindex: read fwd length: %w", err)
	}
	if fwdLen < 0 {
		return nil, fmt.Errorf("biindex: invalid fwd length %d", fwdLen)
	}
	fwdBytes := make([]byte, fwdLen)
	if _, err := io.ReadFull(br, fwdBytes); err != nil {
		return nil, fmt.Errorf("biindex: read fwd bytes: %w", err)
	}
	fwdIdx, err := fmindex.ReadFrom(bytes.NewReader(fwdBytes))
	if err != nil {
		return nil, fmt.Errorf("biindex: decode fwd: %w", err)
	}

	var revLen int64
	if err := binary.Read(br, binary.LittleEndian, &revLen); err != nil {
		return nil, fmt.Errorf("biindex: read rev length: %w", err)
	}
	if revLen < 0 {
		return nil, fmt.Errorf("biindex: invalid rev length %d", revLen)
	}
	revBytes := make([]byte, revLen)
	if _, err := io.ReadFull(br, revBytes); err != nil {
		return nil, fmt.Errorf("biindex: read rev bytes: %w", err)
	}
	revIdx, err := fmindex.ReadFrom(bytes.NewReader(revBytes))
	if err != nil {
		return nil, fmt.Errorf("biindex: decode rev: %w", err)
	}

	return &BiIndex{fwd: fwdIdx, rev: revIdx}, nil
}

// LoadBi reads a BiIndex from a file.
func LoadBi(path string) (*BiIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open biindex: %w", err)
	}
	defer f.Close()
	return ReadBiFrom(f)
}
