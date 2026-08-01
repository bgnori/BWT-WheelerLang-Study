package bwtsearch

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"index/suffixarray"
	"io"
	"os"
	"unicode/utf8"
)

// stdlibMagic is the on-disk header for indexes built with Go's standard-library
// suffix array.  It must be exactly 7 bytes to match the FM-index magic lengths.
const stdlibMagic = "SAIDX01"

// StdlibIndex wraps Go's standard-library suffix array for full-text search.
// Unlike Index (which is FM-index based), StdlibIndex only supports literal
// (non-regex) patterns.
type StdlibIndex struct {
	text []byte
	sa   *suffixarray.Index
}

// BuildStdlib constructs a StdlibIndex from text.
func BuildStdlib(text []byte) *StdlibIndex {
	t := append([]byte(nil), text...)
	return &StdlibIndex{text: t, sa: suffixarray.New(t)}
}

// BuildStdlibFromFiles concatenates texts with separator and builds a single
// StdlibIndex over the combined corpus.  If separator is nil a newline (\n) is
// used.  The separator must not contain 0x00, for consistency with
// BuildFromFiles and BuildBiFromFiles.
//
// BuildStdlibFromFiles panics if separator contains the byte 0x00.
func BuildStdlibFromFiles(texts [][]byte, separator []byte) *StdlibIndex {
	if separator == nil {
		separator = []byte{'\n'}
	}
	for _, b := range separator {
		if b == 0x00 {
			panic("bwtsearch: separator must not contain 0x00 (reserved as sentinel)")
		}
	}
	if len(texts) == 0 {
		return BuildStdlib(nil)
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
	return BuildStdlib(combined)
}

// TextLen returns the original text length.
// A nil *StdlibIndex returns 0.
func (idx *StdlibIndex) TextLen() int {
	if idx == nil {
		return 0
	}
	return len(idx.text)
}

// Count returns the number of occurrences of pattern in the indexed text.
// A nil *StdlibIndex returns 0.
func (idx *StdlibIndex) Count(pattern []byte) int {
	if idx == nil || idx.sa == nil {
		return 0
	}
	return len(idx.sa.Lookup(pattern, -1))
}

// Locate returns up to limit positions where pattern begins (0-indexed).
// When limit <= 0 all positions are returned.
// A nil *StdlibIndex returns nil.
func (idx *StdlibIndex) Locate(pattern []byte, limit int) []int {
	if idx == nil || idx.sa == nil {
		return nil
	}
	n := limit
	if n <= 0 {
		n = -1
	}
	positions := idx.sa.Lookup(pattern, n)
	out := make([]int, len(positions))
	copy(out, positions)
	return out
}

// ContextAround returns a human-readable snippet of text centred on position
// pos, showing ctxSize bytes on each side plus patLen bytes of the match itself.
// A nil *StdlibIndex returns the empty string.
func (idx *StdlibIndex) ContextAround(pos, patLen, ctxSize int) string {
	if idx == nil {
		return ""
	}
	n := len(idx.text)
	start := pos - ctxSize
	if start < 0 {
		start = 0
	}
	end := pos + patLen + ctxSize
	if end > n {
		end = n
	}
	for start > 0 && !utf8.RuneStart(idx.text[start]) {
		start--
	}
	for end < n && !utf8.RuneStart(idx.text[end]) {
		end++
	}
	return string(idx.text[start:end])
}

// WriteTo serialises the index to w in the SAIDX01 format.
// It implements io.WriterTo.
// A nil *StdlibIndex returns an error.
func (idx *StdlibIndex) WriteTo(w io.Writer) (int64, error) {
	if idx == nil || idx.sa == nil {
		return 0, fmt.Errorf("nil stdlib index")
	}
	cw := &saCountingWriter{w: w}
	bw := bufio.NewWriterSize(cw, 1<<20)

	if _, err := bw.WriteString(stdlibMagic); err != nil {
		return cw.n, err
	}
	if err := binary.Write(bw, binary.LittleEndian, int64(len(idx.text))); err != nil {
		return cw.n, err
	}
	if _, err := bw.Write(idx.text); err != nil {
		return cw.n, err
	}
	if err := bw.Flush(); err != nil {
		return cw.n, err
	}
	if err := idx.sa.Write(cw); err != nil {
		return cw.n, err
	}
	return cw.n, nil
}

// Save writes the index to path.
func (idx *StdlibIndex) Save(path string) error {
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

// ReadStdlibFrom deserialises a StdlibIndex from r.
// The reader must be positioned at the start of a SAIDX01 stream.
func ReadStdlibFrom(r io.Reader) (*StdlibIndex, error) {
	br := bufio.NewReaderSize(r, 1<<20)

	hdr := make([]byte, len(stdlibMagic))
	if _, err := io.ReadFull(br, hdr); err != nil {
		return nil, fmt.Errorf("stdlibindex: read magic: %w", err)
	}
	if string(hdr) != stdlibMagic {
		return nil, fmt.Errorf("stdlibindex: bad magic %q (want %q)", string(hdr), stdlibMagic)
	}

	var tlen int64
	if err := binary.Read(br, binary.LittleEndian, &tlen); err != nil {
		return nil, fmt.Errorf("stdlibindex: read text len: %w", err)
	}
	if tlen < 0 {
		return nil, fmt.Errorf("stdlibindex: invalid text length %d", tlen)
	}
	text := make([]byte, tlen)
	if _, err := io.ReadFull(br, text); err != nil {
		return nil, fmt.Errorf("stdlibindex: read text: %w", err)
	}

	sa := new(suffixarray.Index)
	if err := sa.Read(br); err != nil {
		return nil, fmt.Errorf("stdlibindex: read SA: %w", err)
	}

	return &StdlibIndex{text: text, sa: sa}, nil
}

// LoadStdlib reads a StdlibIndex from a file.
func LoadStdlib(path string) (*StdlibIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open index: %w", err)
	}
	defer f.Close()
	return ReadStdlibFrom(f)
}

// saCountingWriter wraps an io.Writer and tracks the total bytes written.
type saCountingWriter struct {
	w io.Writer
	n int64
}

func (cw *saCountingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)
	return n, err
}
