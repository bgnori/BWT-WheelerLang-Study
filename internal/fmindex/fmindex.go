// Package fmindex implements an FM-index, a compressed full-text index that
// forms a Wheeler graph over the text.
//
// # Wheeler graph structure
//
// A Wheeler graph is a directed labeled graph whose nodes can be linearly
// ordered (the Wheeler order) such that:
//   - Edges with smaller labels precede edges with larger labels in the order.
//   - For edges sharing the same label, source order is preserved in dest order.
//
// The FM-index realises a Wheeler graph for a string T as follows:
//   - Nodes: positions in the F column of the BWT matrix (sorted suffixes).
//   - Edges: labeled by the BWT character (L column).
//   - Wheeler order: the lexicographic order of the sorted suffixes.
//
// The index stores:
//   - BWT string (L column)
//   - Suffix array (SA) for text position look-up
//   - C[c]     – cumulative count of chars strictly less than c in T
//   - Occ[c,i] – number of occurrences of c in BWT[0..i-1]  (bit-vector rank)
//
// Backward search over this structure answers pattern queries in O(|P|) time.
package fmindex

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/bgnori/bwt-wheelerlang-study/internal/bitvector"
)

// sentinel is appended to the text before indexing.  It must be the
// lexicographically smallest byte value so that the sentinel suffix always
// sorts first in the suffix array.
const sentinel = byte(0)

// Index is a fully-functional FM-index (Wheeler graph) for byte strings.
type Index struct {
	n    int          // len(text) + 1  (includes sentinel)
	text []byte       // original text  (without sentinel)
	bwt  []byte       // Burrows-Wheeler Transform
	sa   []int32      // suffix array
	c    [256]int     // C array
	occ  occStructure // occurrence array (bitvectors or wavelet tree)
	algo SuffixArrayAlgorithm
	typ  OccStructure
	rope *rope
}

// SuffixArrayAlgorithm selects the suffix-array construction algorithm.
type SuffixArrayAlgorithm int

const (
	// AlgorithmDoubling uses the prefix-doubling (Manber-Myers) algorithm
	// with O(n log² n) time. This is the default.
	AlgorithmDoubling SuffixArrayAlgorithm = iota
	// AlgorithmSAIS uses the SA-IS (Suffix Array – Induced Sorting) algorithm
	// by Nong, Zhang & Chan with O(n) time.
	AlgorithmSAIS
)

// Build constructs an FM-index from text using the default doubling algorithm.
// The text must not contain the null byte (0x00); it is reserved as sentinel.
func Build(text []byte) *Index {
	return BuildWithOptions(text, AlgorithmDoubling, OccBitvectors)
}

// BuildWithAlgorithm constructs an FM-index from text using the specified
// suffix-array construction algorithm.
// The text must not contain the null byte (0x00); it is reserved as sentinel.
func BuildWithAlgorithm(text []byte, algo SuffixArrayAlgorithm) *Index {
	return BuildWithOptions(text, algo, OccBitvectors)
}

// BuildWithOptions constructs an FM-index with an explicit suffix-array
// construction algorithm and an explicit occurrence-array structure.
// The text must not contain the null byte (0x00); it is reserved as sentinel.
func BuildWithOptions(text []byte, algo SuffixArrayAlgorithm, occType OccStructure) *Index {
	n, bwt, sa32, c, occ := buildState(text, algo, occType)
	return &Index{
		n:    n,
		text: append([]byte(nil), text...),
		bwt:  bwt,
		sa:   sa32,
		c:    c,
		occ:  occ,
		algo: algo,
		typ:  occType,
		rope: newRopeFromBytes(text),
	}
}

func buildState(text []byte, algo SuffixArrayAlgorithm, occType OccStructure) (int, []byte, []int32, [256]int, occStructure) {
	// --- 1. Append sentinel -------------------------------------------------
	n := len(text) + 1
	t := make([]byte, n)
	copy(t, text)
	t[n-1] = sentinel

	// --- 2. Suffix array ----------------------------------------------------
	var sa []int
	switch algo {
	case AlgorithmSAIS:
		sa = buildSuffixArraySAIS(t)
	default:
		sa = buildSuffixArray(t)
	}

	// --- 3. BWT  (L column = character preceding each sorted suffix) --------
	bwt := make([]byte, n)
	for i, pos := range sa {
		if pos == 0 {
			bwt[i] = t[n-1]
		} else {
			bwt[i] = t[pos-1]
		}
	}

	// --- 4. C array ---------------------------------------------------------
	var freq [256]int
	for _, b := range t {
		freq[b]++
	}
	var c [256]int
	total := 0
	for i := range c {
		c[i] = total
		total += freq[i]
	}

	// --- 5. Occurrence array ------------------------------------------------
	occ := buildOcc(bwt, occType)

	// Convert SA to int32 to halve memory on 64-bit systems
	sa32 := make([]int32, n)
	for i, v := range sa {
		sa32[i] = int32(v)
	}
	return n, bwt, sa32, c, occ
}

// --- Public query methods ---------------------------------------------------

// TextLen returns the length of the indexed text (without sentinel).
func (idx *Index) TextLen() int { return idx.n - 1 }

// SALen returns the full length of the suffix array (text + sentinel).
func (idx *Index) SALen() int { return idx.n }

// SAAt returns the text position stored at suffix-array index i.
func (idx *Index) SAAt(i int) int { return int(idx.sa[i]) }

// OccCount returns the number of occurrences of byte b in BWT[0..i-1].
func (idx *Index) OccCount(b byte, i int) int {
	return idx.occ.rank(b, i)
}

// CValue returns C[b]: the number of suffixes whose first character is
// lexicographically smaller than b.
func (idx *Index) CValue(b byte) int { return idx.c[b] }

// BackwardSearchStep narrows the SA interval [lo, hi) by one character b
// to the left.  Returns (newLo, newHi); newLo >= newHi means not found.
func (idx *Index) BackwardSearchStep(b byte, lo, hi int) (int, int) {
	newLo := idx.c[b] + idx.OccCount(b, lo)
	newHi := idx.c[b] + idx.OccCount(b, hi)
	return newLo, newHi
}

// BackwardSearch returns the SA interval [lo, hi) for the byte pattern p.
// All positions sa[lo..hi-1] in the text are starting points of p.
// Returns lo >= hi when p is absent.
func (idx *Index) BackwardSearch(p []byte) (lo, hi int) {
	lo, hi = 0, idx.n
	for i := len(p) - 1; i >= 0 && lo < hi; i-- {
		lo, hi = idx.BackwardSearchStep(p[i], lo, hi)
	}
	return
}

// Count returns the number of occurrences of p in the indexed text.
func (idx *Index) Count(p []byte) int {
	lo, hi := idx.BackwardSearch(p)
	if lo >= hi {
		return 0
	}
	return hi - lo
}

// Locate returns up to limit text positions where p begins (0-indexed).
// When limit <= 0 all positions are returned.
func (idx *Index) Locate(p []byte, limit int) []int {
	lo, hi := idx.BackwardSearch(p)
	if lo >= hi {
		return nil
	}
	count := hi - lo
	if limit > 0 && count > limit {
		count = limit
	}
	out := make([]int, count)
	for i := range out {
		out[i] = int(idx.sa[lo+i])
	}
	return out
}

// ContextAround returns a human-readable snippet of text centred on position
// pos, showing ctxSize bytes on each side plus patLen bytes of the match
// itself.  Start and end positions are adjusted outward to avoid splitting
// multi-byte UTF-8 characters.
func (idx *Index) ContextAround(pos, patLen, ctxSize int) string {
	n := len(idx.text)
	start := pos - ctxSize
	if start < 0 {
		start = 0
	}
	end := pos + patLen + ctxSize
	if end > n {
		end = n
	}
	// Align start to the beginning of a UTF-8 rune so we don't return a
	// partial multi-byte sequence before the match.
	for start > 0 && !utf8.RuneStart(idx.text[start]) {
		start--
	}
	// Align end to the start of the next rune (or end of text) so we don't
	// return a partial multi-byte sequence after the match.
	for end < n && !utf8.RuneStart(idx.text[end]) {
		end++
	}
	return string(idx.text[start:end])
}

// BWT returns a copy of the Burrows-Wheeler Transform.
func (idx *Index) BWT() []byte { return append([]byte(nil), idx.bwt...) }

// SuffixAt returns up to maxLen characters of the suffix at SA position i.
func (idx *Index) SuffixAt(saPos, maxLen int) string {
	textPos := int(idx.sa[saPos])
	end := textPos + maxLen
	if end > len(idx.text) {
		end = len(idx.text)
	}
	return string(idx.text[textPos:end])
}

// AlphabetSize returns the number of distinct characters present in the text.
func (idx *Index) AlphabetSize() int {
	var seen [256]bool
	for _, b := range idx.bwt {
		seen[b] = true
	}
	count := 0
	for _, s := range seen {
		if s {
			count++
		}
	}
	return count
}

// WheelerGraphMermaid returns a Mermaid flowchart representation of the
// Wheeler graph encoded by this FM-index.
//
// maxNodes limits the number of nodes emitted in Wheeler order; values <= 0
// include all nodes.
func (idx *Index) WheelerGraphMermaid(maxNodes int) string {
	n := idx.SALen()
	if maxNodes <= 0 || maxNodes > n {
		maxNodes = n
	}

	var b strings.Builder
	b.WriteString("flowchart LR\n")
	b.WriteString("  %% Node order is the Wheeler order (SA rank).\n")

	for i := 0; i < maxNodes; i++ {
		suf := idx.suffixPreview(i, 12)
		fmt.Fprintf(&b, "  n%d[\"%d: SA=%d %s\"]\n", i, i, idx.SAAt(i), quoteForMermaid(suf))
	}

	hiddenEdges := 0
	for i := 0; i < maxNodes; i++ {
		label := idx.bwt[i]
		to := idx.c[label] + idx.OccCount(label, i)
		if to >= maxNodes {
			hiddenEdges++
			continue
		}
		fmt.Fprintf(&b, "  n%d --\"%s\"--> n%d\n", i, edgeLabel(label), to)
	}

	if maxNodes < n {
		hiddenNodes := n - maxNodes
		fmt.Fprintf(&b, "  hidden[(\"... %d more nodes omitted\")]\n", hiddenNodes)
		if hiddenEdges > 0 {
			fmt.Fprintf(&b, "  note[\"%d edges to omitted nodes\"]\n", hiddenEdges)
			b.WriteString("  note -.-> hidden\n")
		}
	}

	return b.String()
}

func (idx *Index) suffixPreview(saPos, maxLen int) string {
	textPos := idx.SAAt(saPos)
	if textPos >= len(idx.text) {
		return "$"
	}
	end := textPos + maxLen
	if end > len(idx.text) {
		end = len(idx.text)
	}
	return string(idx.text[textPos:end]) + "$"
}

func edgeLabel(b byte) string {
	if b == sentinel {
		return "$"
	}
	if b >= 32 && b <= 126 && b != '"' {
		return string([]byte{b})
	}
	return fmt.Sprintf("0x%02X", b)
}

func quoteForMermaid(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\\\"")
}

// --- Suffix array construction ---------------------------------------------

// buildSuffixArray builds the suffix array of text using the prefix-doubling
// (Manber-Myers) algorithm in O(n log² n) time.
// text must end with byte 0 (sentinel) which is the unique minimum character.
func buildSuffixArray(text []byte) []int {
	n := len(text)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return []int{0}
	}

	sa := make([]int, n)
	rank := make([]int, n)
	tmp := make([]int, n)

	for i := range sa {
		sa[i] = i
		rank[i] = int(text[i])
	}

	for gap := 1; gap < n; gap *= 2 {
		r := rank // capture for closure
		g := gap
		sort.Slice(sa, func(a, b int) bool {
			ia, ib := sa[a], sa[b]
			if r[ia] != r[ib] {
				return r[ia] < r[ib]
			}
			r2a, r2b := -1, -1
			if ia+g < n {
				r2a = r[ia+g]
			}
			if ib+g < n {
				r2b = r[ib+g]
			}
			return r2a < r2b
		})

		tmp[sa[0]] = 0
		for i := 1; i < n; i++ {
			prev, curr := sa[i-1], sa[i]
			tmp[curr] = tmp[prev]
			if r[curr] != r[prev] {
				tmp[curr]++
			} else {
				r2c, r2p := -1, -1
				if curr+g < n {
					r2c = r[curr+g]
				}
				if prev+g < n {
					r2p = r[prev+g]
				}
				if r2c != r2p {
					tmp[curr]++
				}
			}
		}
		copy(rank, tmp)
		if rank[sa[n-1]] == n-1 {
			break // all ranks unique → SA is complete
		}
	}
	return sa
}

// --- Persistence -----------------------------------------------------------

const magic = "FMIDX01"   // bitvector-based occ
const magicV2 = "FMIDX02" // wavelet-tree-based occ (occ rebuilt from BWT on load)

// countingWriter wraps an io.Writer and tracks the total bytes written.
type countingWriter struct {
	w io.Writer
	n int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)
	return n, err
}

// WriteTo serialises the index to w.
// Bitvector-based indexes use the FMIDX01 format (backward compatible).
// Wavelet-tree-based indexes use the FMIDX02 format.
// It implements io.WriterTo.
func (idx *Index) WriteTo(w io.Writer) (int64, error) {
	if _, ok := idx.occ.(*waveletOcc); ok {
		return idx.writeToV2(w)
	}
	return idx.writeToV1(w)
}

// writeToV1 writes the FMIDX01 (bitvector) format.
func (idx *Index) writeToV1(w io.Writer) (int64, error) {
	cw := &countingWriter{w: w}
	bw := bufio.NewWriterSize(cw, 1<<20)

	if _, err := bw.WriteString(magic); err != nil {
		return 0, err
	}
	if err := binary.Write(bw, binary.LittleEndian, int64(idx.n)); err != nil {
		return 0, err
	}

	// original text
	if err := binary.Write(bw, binary.LittleEndian, int64(len(idx.text))); err != nil {
		return 0, err
	}
	if _, err := bw.Write(idx.text); err != nil {
		return 0, err
	}

	// BWT
	if _, err := bw.Write(idx.bwt); err != nil {
		return 0, err
	}

	// SA as packed int32
	saBuf := make([]byte, len(idx.sa)*4)
	for i, v := range idx.sa {
		binary.LittleEndian.PutUint32(saBuf[i*4:], uint32(v))
	}
	if _, err := bw.Write(saBuf); err != nil {
		return 0, err
	}

	// C array (256 × int64)
	cBuf := make([]byte, 256*8)
	for i, v := range idx.c {
		binary.LittleEndian.PutUint64(cBuf[i*8:], uint64(v))
	}
	if _, err := bw.Write(cBuf); err != nil {
		return 0, err
	}

	// Occ bit-vectors
	bitvecs := idx.occ.(*bitvecOcc).vecs
	for i := 0; i < 256; i++ {
		if bitvecs[i] != nil {
			if err := bw.WriteByte(1); err != nil {
				return 0, err
			}
			if _, err := bitvecs[i].WriteTo(bw); err != nil {
				return 0, err
			}
		} else {
			if err := bw.WriteByte(0); err != nil {
				return 0, err
			}
		}
	}

	if err := bw.Flush(); err != nil {
		return cw.n, err
	}
	return cw.n, nil
}

// writeToV2 writes the FMIDX02 (wavelet tree) format.
// The occurrence array is NOT written; it is rebuilt from the BWT on load.
func (idx *Index) writeToV2(w io.Writer) (int64, error) {
	cw := &countingWriter{w: w}
	bw := bufio.NewWriterSize(cw, 1<<20)

	if _, err := bw.WriteString(magicV2); err != nil {
		return 0, err
	}
	if err := binary.Write(bw, binary.LittleEndian, int64(idx.n)); err != nil {
		return 0, err
	}

	// original text
	if err := binary.Write(bw, binary.LittleEndian, int64(len(idx.text))); err != nil {
		return 0, err
	}
	if _, err := bw.Write(idx.text); err != nil {
		return 0, err
	}

	// BWT
	if _, err := bw.Write(idx.bwt); err != nil {
		return 0, err
	}

	// SA as packed int32
	saBuf := make([]byte, len(idx.sa)*4)
	for i, v := range idx.sa {
		binary.LittleEndian.PutUint32(saBuf[i*4:], uint32(v))
	}
	if _, err := bw.Write(saBuf); err != nil {
		return 0, err
	}

	// C array (256 × int64)
	cBuf := make([]byte, 256*8)
	for i, v := range idx.c {
		binary.LittleEndian.PutUint64(cBuf[i*8:], uint64(v))
	}
	if _, err := bw.Write(cBuf); err != nil {
		return 0, err
	}

	// (No occ section – reconstructed from BWT on load.)

	if err := bw.Flush(); err != nil {
		return cw.n, err
	}
	return cw.n, nil
}

// ReadFrom deserialises an index from r.
// Both the FMIDX01 (bitvectors) and FMIDX02 (wavelet tree) formats are
// supported.
func ReadFrom(r io.Reader) (*Index, error) {
	br := bufio.NewReaderSize(r, 1<<20)

	hdr := make([]byte, len(magic)) // both magics have the same length
	if _, err := io.ReadFull(br, hdr); err != nil {
		return nil, fmt.Errorf("fmindex: read magic: %w", err)
	}
	switch string(hdr) {
	case magic:
		return readFromV1(br)
	case magicV2:
		return readFromV2(br)
	default:
		return nil, fmt.Errorf("fmindex: bad magic %q (want %q or %q)", hdr, magic, magicV2)
	}
}

// Append incrementally extends the indexed text and rebuilds FM-index state.
// The original suffix-array algorithm and occurrence structure are preserved.
func (idx *Index) Append(text []byte) {
	if idx == nil || len(text) == 0 {
		return
	}
	if idx.rope == nil {
		idx.rope = newRopeFromBytes(idx.text)
	}
	idx.rope = idx.rope.Append(text)
	combined := idx.rope.Bytes()
	n, bwt, sa32, c, occ := buildState(combined, idx.algo, idx.typ)
	idx.n = n
	idx.text = combined
	idx.bwt = bwt
	idx.sa = sa32
	idx.c = c
	idx.occ = occ
}

// readCommonHeader reads n, text, bwt, sa, and c — fields shared by both formats.
func readCommonHeader(br *bufio.Reader) (*Index, error) {
	idx := &Index{}

	var n64 int64
	if err := binary.Read(br, binary.LittleEndian, &n64); err != nil {
		return nil, fmt.Errorf("fmindex: read n: %w", err)
	}
	idx.n = int(n64)

	var tlen int64
	if err := binary.Read(br, binary.LittleEndian, &tlen); err != nil {
		return nil, fmt.Errorf("fmindex: read text len: %w", err)
	}
	idx.text = make([]byte, tlen)
	if _, err := io.ReadFull(br, idx.text); err != nil {
		return nil, fmt.Errorf("fmindex: read text: %w", err)
	}

	idx.bwt = make([]byte, idx.n)
	if _, err := io.ReadFull(br, idx.bwt); err != nil {
		return nil, fmt.Errorf("fmindex: read BWT: %w", err)
	}

	saBuf := make([]byte, idx.n*4)
	if _, err := io.ReadFull(br, saBuf); err != nil {
		return nil, fmt.Errorf("fmindex: read SA: %w", err)
	}
	idx.sa = make([]int32, idx.n)
	for i := range idx.sa {
		idx.sa[i] = int32(binary.LittleEndian.Uint32(saBuf[i*4:]))
	}

	cBuf := make([]byte, 256*8)
	if _, err := io.ReadFull(br, cBuf); err != nil {
		return nil, fmt.Errorf("fmindex: read C: %w", err)
	}
	for i := range idx.c {
		idx.c[i] = int(binary.LittleEndian.Uint64(cBuf[i*8:]))
	}

	return idx, nil
}

// readFromV1 reads the FMIDX01 (bitvector) format (magic already consumed).
func readFromV1(br *bufio.Reader) (*Index, error) {
	idx, err := readCommonHeader(br)
	if err != nil {
		return nil, err
	}

	occ := &bitvecOcc{}
	for i := 0; i < 256; i++ {
		present, err := br.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("fmindex: read occ present[%d]: %w", i, err)
		}
		if present != 0 {
			bv, err := bitvector.ReadFrom(br)
			if err != nil {
				return nil, fmt.Errorf("fmindex: read occ[%d]: %w", i, err)
			}
			occ.vecs[i] = bv
		}
	}
	idx.occ = occ
	idx.algo = AlgorithmDoubling
	idx.typ = OccBitvectors
	idx.rope = newRopeFromBytes(idx.text)
	return idx, nil
}

// readFromV2 reads the FMIDX02 (wavelet tree) format (magic already consumed).
// The occurrence array is reconstructed from the stored BWT.
func readFromV2(br *bufio.Reader) (*Index, error) {
	idx, err := readCommonHeader(br)
	if err != nil {
		return nil, err
	}
	idx.occ = buildOcc(idx.bwt, OccWaveletTree)
	idx.algo = AlgorithmDoubling
	idx.typ = OccWaveletTree
	idx.rope = newRopeFromBytes(idx.text)
	return idx, nil
}
