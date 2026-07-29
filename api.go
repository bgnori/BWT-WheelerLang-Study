// Package bwtsearch provides a stable public API for building and querying
// FM-index based full-text search indexes.
package bwtsearch

import (
	"fmt"
	"io"
	"os"

	"github.com/bgnori/bwt-wheelerlang-study/internal/fmindex"
	"github.com/bgnori/bwt-wheelerlang-study/internal/starfree"
)

// SuffixArrayAlgorithm selects the suffix-array construction algorithm.
type SuffixArrayAlgorithm int

const (
	// AlgorithmDoubling uses the prefix-doubling algorithm.
	AlgorithmDoubling SuffixArrayAlgorithm = SuffixArrayAlgorithm(fmindex.AlgorithmDoubling)
	// AlgorithmSAIS uses the SA-IS algorithm.
	AlgorithmSAIS SuffixArrayAlgorithm = SuffixArrayAlgorithm(fmindex.AlgorithmSAIS)
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

// Build constructs an index from text using the default algorithm.
func Build(text []byte) *Index {
	return &Index{inner: fmindex.Build(text)}
}

// BuildWithAlgorithm constructs an index from text with an explicit algorithm.
func BuildWithAlgorithm(text []byte, algo SuffixArrayAlgorithm) *Index {
	return &Index{inner: fmindex.BuildWithAlgorithm(text, fmindex.SuffixArrayAlgorithm(algo))}
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
func (idx *Index) TextLen() int { return idx.inner.TextLen() }

// SALen returns the suffix-array length (text + sentinel).
func (idx *Index) SALen() int { return idx.inner.SALen() }

// SAAt returns the text position stored at suffix-array index i.
func (idx *Index) SAAt(i int) int { return idx.inner.SAAt(i) }

// AlphabetSize returns the number of distinct characters in the text.
func (idx *Index) AlphabetSize() int { return idx.inner.AlphabetSize() }

// BWT returns a copy of the Burrows-Wheeler Transform.
func (idx *Index) BWT() []byte { return idx.inner.BWT() }

// Count returns the number of occurrences of pattern.
func (idx *Index) Count(pattern []byte) int { return idx.inner.Count(pattern) }

// Locate returns up to limit positions where pattern begins.
func (idx *Index) Locate(pattern []byte, limit int) []int { return idx.inner.Locate(pattern, limit) }

// ContextAround returns a snippet around a match position.
func (idx *Index) ContextAround(pos, patLen, ctxSize int) string {
	return idx.inner.ContextAround(pos, patLen, ctxSize)
}

// WheelerGraphMermaid returns a Mermaid graph representation.
func (idx *Index) WheelerGraphMermaid(maxNodes int) string {
	return idx.inner.WheelerGraphMermaid(maxNodes)
}

// Check validates that pattern is a star-free regular expression.
func Check(pattern string) error {
	return starfree.Check(pattern)
}

// Search runs a star-free regex search over idx.
func Search(idx *Index, pattern string, limit int) (*SearchResult, error) {
	if idx == nil || idx.inner == nil {
		return nil, fmt.Errorf("nil index")
	}

	res, err := starfree.Search(idx.inner, pattern, limit)
	if err != nil {
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
