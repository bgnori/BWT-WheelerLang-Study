// Package rindex implements a run-length BWT (RLBWT) based occurrence structure.
//
// The BWT is stored as a compact sequence of (character, length) runs rather
// than as a full flat byte array.  This is the basis of the r-index family of
// compressed indexes, which are space-efficient for repetitive texts.
//
// Rank queries are answered in O(log r) time where r is the number of BWT runs,
// by binary searching per-character run tables.
package rindex

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Run is a single BWT run: Char repeated Len times consecutively.
type Run struct {
	Char byte
	Len  int32
}

// charRunEntry records one run of a given character for O(log r) rank queries.
type charRunEntry struct {
	start     int // start position of this run in the BWT
	length    int // length of this run
	cumBefore int // total occurrences of this char in all preceding runs
}

// RLBWT stores the run-length encoded BWT and supports O(log r) rank queries.
type RLBWT struct {
	runs     []Run
	nTotal   int // total BWT length (sum of all run lengths)
	charRuns [256][]charRunEntry
}

// Build constructs an RLBWT from the BWT byte sequence.
func Build(bwt []byte) *RLBWT {
	if len(bwt) == 0 {
		return &RLBWT{}
	}

	// Compress into runs.
	var runs []Run
	i := 0
	for i < len(bwt) {
		j := i + 1
		for j < len(bwt) && bwt[j] == bwt[i] {
			j++
		}
		runs = append(runs, Run{Char: bwt[i], Len: int32(j - i)})
		i = j
	}

	// Build per-character run entry tables for fast rank queries.
	var charRuns [256][]charRunEntry
	cumCounts := [256]int{}
	pos := 0
	for _, r := range runs {
		b := r.Char
		charRuns[b] = append(charRuns[b], charRunEntry{
			start:     pos,
			length:    int(r.Len),
			cumBefore: cumCounts[b],
		})
		cumCounts[b] += int(r.Len)
		pos += int(r.Len)
	}

	return &RLBWT{
		runs:     runs,
		nTotal:   len(bwt),
		charRuns: charRuns,
	}
}

// Rank returns the number of occurrences of b in bwt[0..i).
func (rl *RLBWT) Rank(b byte, i int) int {
	if i <= 0 || rl.nTotal == 0 {
		return 0
	}
	if i > rl.nTotal {
		i = rl.nTotal
	}
	entries := rl.charRuns[b]
	if len(entries) == 0 {
		return 0
	}

	// Binary search: find the last entry whose run starts before position i.
	lo, hi := 0, len(entries)
	for lo < hi {
		mid := (lo + hi) / 2
		if entries[mid].start < i {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == 0 {
		return 0
	}
	e := entries[lo-1]
	// Count the full preceding runs.
	count := e.cumBefore
	// Add the portion of e's own run that falls within [e.start, i).
	inRun := i - e.start
	if inRun > e.length {
		inRun = e.length
	}
	return count + inRun
}

// NumRuns returns the number of BWT runs.
func (rl *RLBWT) NumRuns() int { return len(rl.runs) }

// WriteTo serialises the RLBWT to w.
// Format: nTotal (int64), numRuns (int64), runs as (byte, int32) pairs.
func (rl *RLBWT) WriteTo(w io.Writer) (int64, error) {
	var written int64
	if err := binary.Write(w, binary.LittleEndian, int64(rl.nTotal)); err != nil {
		return 0, fmt.Errorf("rlbwt: write nTotal: %w", err)
	}
	written += 8
	if err := binary.Write(w, binary.LittleEndian, int64(len(rl.runs))); err != nil {
		return written, fmt.Errorf("rlbwt: write numRuns: %w", err)
	}
	written += 8
	for _, r := range rl.runs {
		if err := binary.Write(w, binary.LittleEndian, r.Char); err != nil {
			return written, fmt.Errorf("rlbwt: write run char: %w", err)
		}
		written++
		if err := binary.Write(w, binary.LittleEndian, r.Len); err != nil {
			return written, fmt.Errorf("rlbwt: write run len: %w", err)
		}
		written += 4
	}
	return written, nil
}

// ReadFrom deserialises an RLBWT from r and rebuilds the per-character run tables.
func ReadFrom(r io.Reader) (*RLBWT, error) {
	var nTotal64, numRuns64 int64
	if err := binary.Read(r, binary.LittleEndian, &nTotal64); err != nil {
		return nil, fmt.Errorf("rlbwt: read nTotal: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &numRuns64); err != nil {
		return nil, fmt.Errorf("rlbwt: read numRuns: %w", err)
	}
	runs := make([]Run, numRuns64)
	for i := range runs {
		if err := binary.Read(r, binary.LittleEndian, &runs[i].Char); err != nil {
			return nil, fmt.Errorf("rlbwt: read run[%d] char: %w", i, err)
		}
		if err := binary.Read(r, binary.LittleEndian, &runs[i].Len); err != nil {
			return nil, fmt.Errorf("rlbwt: read run[%d] len: %w", i, err)
		}
	}

	// Rebuild per-character run tables.
	var charRuns [256][]charRunEntry
	cumCounts := [256]int{}
	pos := 0
	for _, run := range runs {
		b := run.Char
		charRuns[b] = append(charRuns[b], charRunEntry{
			start:     pos,
			length:    int(run.Len),
			cumBefore: cumCounts[b],
		})
		cumCounts[b] += int(run.Len)
		pos += int(run.Len)
	}

	return &RLBWT{
		runs:     runs,
		nTotal:   int(nTotal64),
		charRuns: charRuns,
	}, nil
}
