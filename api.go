// Package bwtsearch provides a stable public API for building and querying
// FM-index based full-text search indexes.
package bwtsearch

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bgnori/textindex/internal/fmindex"
	"github.com/bgnori/textindex/internal/starfree"
)

// ViolationError is returned by Check and Search when the pattern contains a
// construct that violates the star-free constraint (e.g. Kleene star, +, or
// unbounded repetition).  Callers can use errors.As to inspect the details.
type ViolationError struct {
	// Op is a human-readable name of the offending operator (e.g. "Kleene star (*)").
	Op string
	// SubExpr is the string form of the offending sub-expression.
	SubExpr string
}

func (e *ViolationError) Error() string {
	return fmt.Sprintf(
		"star-free violation: %q uses %s, which introduces unbounded iteration",
		e.SubExpr, e.Op,
	)
}

// UnsupportedError is returned by Check and Search when the pattern contains a
// regex construct that this FM-index matcher does not support (for example,
// position anchors).
type UnsupportedError struct {
	// Op is a human-readable name of the unsupported operator.
	Op string
	// SubExpr is the string form of the offending sub-expression.
	SubExpr string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf(
		"unsupported regex construct: %q uses %s, which cannot be evaluated by FM-index backward search",
		e.SubExpr, e.Op,
	)
}

// SuffixArrayAlgorithm selects the suffix-array construction algorithm.
type SuffixArrayAlgorithm int

const (
	// AlgorithmDoubling uses the prefix-doubling algorithm.
	AlgorithmDoubling SuffixArrayAlgorithm = SuffixArrayAlgorithm(fmindex.AlgorithmDoubling)
	// AlgorithmSAIS uses the SA-IS algorithm.
	AlgorithmSAIS SuffixArrayAlgorithm = SuffixArrayAlgorithm(fmindex.AlgorithmSAIS)
)

// OccStructure selects the occurrence-array implementation inside the FM-index.
type OccStructure int

const (
	// OccBitvectors uses one succinct bit-vector per distinct character
	// representation. Indexes built with this option are written in the FMIDX05
	// on-disk format.
	OccBitvectors OccStructure = OccStructure(fmindex.OccBitvectors)
	// OccWaveletTree uses a Wavelet Tree over the BWT, providing O(log σ)
	// rank queries and O(n log σ) total space.  This is advantageous when
	// the alphabet is large or nearly all 256 byte values appear.
	// Indexes built with this option are written in the FMIDX06 on-disk
	// format.
	OccWaveletTree OccStructure = OccStructure(fmindex.OccWaveletTree)
	// OccWaveletMatrix uses a Wavelet Matrix over the BWT.  It provides the
	// same O(log σ) rank complexity as OccWaveletTree but with a flat,
	// cache-friendly memory layout.  Indexes built with this option use the
	// FMIDX07 on-disk format.
	OccWaveletMatrix OccStructure = OccStructure(fmindex.OccWaveletMatrix)
	// OccRLBWT uses a run-length encoded BWT for rank queries.  The BWT is
	// stored as a compact sequence of equal-character runs, which is the
	// foundation of r-index style compressed indexes.  Rank queries run in
	// O(log r) time where r is the number of BWT runs.  This is the default
	// space-efficient for highly repetitive texts.
	// Indexes built with this option use the FMIDX08 on-disk format.
	OccRLBWT OccStructure = OccStructure(fmindex.OccRLBWT)
	// OccRRR uses one Raman-Raman-Rao (RRR) bit-vector per distinct character
	// over the BWT. Indexes built with this option use the FMIDX09 on-disk
	// format.
	OccRRR OccStructure = OccStructure(fmindex.OccRRR)
	// OccEliasFano uses one Elias-Fano encoded position list per distinct
	// character over the BWT. Indexes built with this option use the FMIDX10
	// on-disk format.
	OccEliasFano OccStructure = OccStructure(fmindex.OccEliasFano)
	// OccPoppy uses one interleaved RRR (Poppy-style) bit-vector per distinct
	// character over the BWT. Indexes built with this option use the FMIDX11
	// on-disk format.
	OccPoppy OccStructure = OccStructure(fmindex.OccPoppy)
	// OccDynamicBitvectors uses one dynamic bit-vector per distinct character
	// over the BWT. Indexes built with this option use the FMIDX12 on-disk
	// format.
	OccDynamicBitvectors OccStructure = OccStructure(fmindex.OccDynamicBitvectors)
)

// Interval is a half-open suffix-array range [Lo, Hi).
type Interval struct {
	Lo int
	Hi int
}

// SearchResult holds the outcome of a star-free regex search.
type SearchResult struct {
	Intervals  []Interval
	TotalCount int
	Truncated  bool
}

// Positions resolves all match positions in the indexed text.
func (sr *SearchResult) Positions(idx *Index) []int {
	if sr == nil || idx == nil || idx.inner == nil {
		return nil
	}
	var out []int
	for _, iv := range sr.Intervals {
		for i := iv.Lo; i < iv.Hi; i++ {
			out = append(out, idx.SAAt(i))
		}
	}
	return out
}

// Index is a wrapper around the internal FM-index implementation.
type Index struct {
	inner *fmindex.Index
}

// Build constructs an index from text using the default SA-IS algorithm and
// RLBWT occurrence structure.
// Passing nil or an empty slice is valid and returns an index over the empty
// text (sentinel only): Count reports 0 for any non-empty pattern, Search
// finds no matches, TextLen is 0, and SALen is 1.
func Build(text []byte) *Index {
	return &Index{inner: fmindex.Build(text)}
}

// BuildFromFiles concatenates texts with separator and builds a single
// FM-index over the combined corpus. If separator is nil a newline (\n) is
// used. The separator must not contain 0x00, which is reserved as the
// FM-index sentinel.
//
// BuildFromFiles panics if separator contains the byte 0x00.
func BuildFromFiles(texts [][]byte, separator []byte) *Index {
	if separator == nil {
		separator = []byte{'\n'}
	}
	for _, b := range separator {
		if b == 0x00 {
			panic("bwtsearch: separator must not contain 0x00 (reserved as sentinel)")
		}
	}
	if len(texts) == 0 {
		return Build(nil)
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
	return Build(combined)
}

// BuildWithAlgorithm constructs an index from text with an explicit algorithm.
func BuildWithAlgorithm(text []byte, algo SuffixArrayAlgorithm) *Index {
	return &Index{inner: fmindex.BuildWithAlgorithm(text, fmindex.SuffixArrayAlgorithm(algo))}
}

// BuildWithOptions constructs an index with an explicit suffix-array
// construction algorithm and an explicit occurrence-array structure.
func BuildWithOptions(text []byte, algo SuffixArrayAlgorithm, occ OccStructure) *Index {
	return &Index{inner: fmindex.BuildWithOptions(text, fmindex.SuffixArrayAlgorithm(algo), fmindex.OccStructure(occ))}
}

// BuildFromFilesWithOptions concatenates texts with separator and builds a
// single FM-index using the specified algorithm and occurrence-array structure.
// If separator is nil a newline (\n) is used. The separator must not contain
// 0x00, which is reserved as the FM-index sentinel.
//
// BuildFromFilesWithOptions panics if separator contains the byte 0x00.
func BuildFromFilesWithOptions(texts [][]byte, separator []byte, algo SuffixArrayAlgorithm, occ OccStructure) *Index {
	if separator == nil {
		separator = []byte{'\n'}
	}
	for _, b := range separator {
		if b == 0x00 {
			panic("bwtsearch: separator must not contain 0x00 (reserved as sentinel)")
		}
	}
	if len(texts) == 0 {
		return BuildWithOptions(nil, algo, occ)
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
	return BuildWithOptions(combined, algo, occ)
}

// ReadFrom deserialises an index from r.
func ReadFrom(r io.Reader) (*Index, error) {
	inner, err := fmindex.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	return &Index{inner: inner}, nil
}

// Load reads an index from a file.
func Load(path string) (*Index, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open index: %w", err)
	}
	defer f.Close()

	return ReadFrom(f)
}

// WriteTo serialises the index to w.
func (idx *Index) WriteTo(w io.Writer) (int64, error) {
	if idx == nil || idx.inner == nil {
		return 0, fmt.Errorf("nil index")
	}
	return idx.inner.WriteTo(w)
}

// Save writes the index to path.
func (idx *Index) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	defer f.Close()

	if _, err := idx.WriteTo(f); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	return nil
}

// TextLen returns the original text length.
// A nil *Index returns 0.
func (idx *Index) TextLen() int {
	if idx == nil || idx.inner == nil {
		return 0
	}
	return idx.inner.TextLen()
}

// SALen returns the suffix-array length (text + sentinel).
// A nil *Index returns 0.
func (idx *Index) SALen() int {
	if idx == nil || idx.inner == nil {
		return 0
	}
	return idx.inner.SALen()
}

// SAAt returns the text position stored at suffix-array index i.
// A nil *Index returns 0.
func (idx *Index) SAAt(i int) int {
	if idx == nil || idx.inner == nil {
		return 0
	}
	return idx.inner.SAAt(i)
}

// AlphabetSize returns the number of distinct characters in the text.
// A nil *Index returns 0.
func (idx *Index) AlphabetSize() int {
	if idx == nil || idx.inner == nil {
		return 0
	}
	return idx.inner.AlphabetSize()
}

// NumBWTRuns returns the number of equal-character runs in the BWT.
// This is the r parameter of the r-index: smaller values indicate more
// repetitive texts and a more compact RLBWT representation.
// A nil *Index returns 0.
func (idx *Index) NumBWTRuns() int {
	if idx == nil || idx.inner == nil {
		return 0
	}
	return idx.inner.NumBWTRuns()
}

// OccType returns the occurrence-array structure used by this index.
// A nil *Index returns OccBitvectors (the zero value).
func (idx *Index) OccType() OccStructure {
	if idx == nil || idx.inner == nil {
		return OccBitvectors
	}
	return OccStructure(idx.inner.OccType())
}

// BWT returns a copy of the Burrows-Wheeler Transform.
// A nil *Index returns nil.
func (idx *Index) BWT() []byte {
	if idx == nil || idx.inner == nil {
		return nil
	}
	return idx.inner.BWT()
}

// Append incrementally appends text to the index (RopeBWT style).
// Existing build options (suffix-array algorithm / occurrence structure) are preserved.
func (idx *Index) Append(text []byte) error {
	if idx == nil || idx.inner == nil {
		return fmt.Errorf("nil index")
	}
	idx.inner.Append(text)
	return nil
}

// Count returns the number of occurrences of pattern.
// A nil *Index returns 0.
func (idx *Index) Count(pattern []byte) int {
	if idx == nil || idx.inner == nil {
		return 0
	}
	return idx.inner.Count(pattern)
}

// Locate returns up to limit positions where pattern begins.
// A nil *Index returns nil.
func (idx *Index) Locate(pattern []byte, limit int) []int {
	if idx == nil || idx.inner == nil {
		return nil
	}
	return idx.inner.Locate(pattern, limit)
}

// ContextAround returns a snippet around a match position.
// A nil *Index returns the empty string.
func (idx *Index) ContextAround(pos, patLen, ctxSize int) string {
	if idx == nil || idx.inner == nil {
		return ""
	}
	return idx.inner.ContextAround(pos, patLen, ctxSize)
}

// WheelerGraphMermaid returns a Mermaid graph representation.
// A nil *Index returns the empty string.
func (idx *Index) WheelerGraphMermaid(maxNodes int) string {
	if idx == nil || idx.inner == nil {
		return ""
	}
	return idx.inner.WheelerGraphMermaid(maxNodes)
}

// Check validates that pattern is a star-free regular expression.
// It returns a *ViolationError if the pattern contains Kleene star, one-or-more,
// or an unbounded repetition; a *UnsupportedError for unsupported constructs
// such as anchors; or a wrapped error for invalid syntax.
func Check(pattern string) error {
	err := starfree.Check(pattern)
	if err == nil {
		return nil
	}
	var sv *starfree.ViolationError
	if errors.As(err, &sv) {
		return &ViolationError{Op: sv.Op, SubExpr: sv.SubExpr}
	}
	var su *starfree.UnsupportedError
	if errors.As(err, &su) {
		return &UnsupportedError{Op: su.Op, SubExpr: su.SubExpr}
	}
	return err
}

// Search runs a star-free regex search over idx.
// It returns a *ViolationError if pattern violates the star-free constraint, or
// a *UnsupportedError for unsupported regex constructs.
func Search(idx *Index, pattern string, limit int) (*SearchResult, error) {
	if idx == nil || idx.inner == nil {
		return nil, fmt.Errorf("nil index")
	}

	res, err := starfree.Search(idx.inner, pattern, limit)
	if err != nil {
		var sv *starfree.ViolationError
		if errors.As(err, &sv) {
			return nil, &ViolationError{Op: sv.Op, SubExpr: sv.SubExpr}
		}
		var su *starfree.UnsupportedError
		if errors.As(err, &su) {
			return nil, &UnsupportedError{Op: su.Op, SubExpr: su.SubExpr}
		}
		return nil, err
	}

	ivs := make([]Interval, len(res.Intervals))
	for i, iv := range res.Intervals {
		ivs[i] = Interval{Lo: iv.Lo, Hi: iv.Hi}
	}

	return &SearchResult{
		Intervals:  ivs,
		TotalCount: res.TotalCount,
		Truncated:  res.Truncated,
	}, nil
}
