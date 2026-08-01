package fmindex

import (
	"github.com/bgnori/textindex/internal/bitvector"
	"github.com/bgnori/textindex/internal/rindex"
	"github.com/bgnori/textindex/internal/wavelet"
	"github.com/bgnori/textindex/internal/waveletmatrix"
)

// OccStructure selects the occurrence-array implementation used inside an
// FM-index.
type OccStructure int

const (
	// OccBitvectors uses one succinct bit-vector per distinct character.
	// Indexes built with this option use the FMIDX05 on-disk format.
	OccBitvectors OccStructure = iota
	// OccWaveletTree uses a Wavelet Tree over the BWT, providing O(log σ)
	// rank queries and O(n log σ) total space.  Indexes built with this
	// option are written in the FMIDX06 on-disk format.
	OccWaveletTree
	// OccWaveletMatrix uses a Wavelet Matrix over the BWT.  It provides the
	// same O(log σ) rank complexity as OccWaveletTree but with a flat,
	// cache-friendly memory layout.  Indexes built with this option use the
	// FMIDX07 on-disk format.
	OccWaveletMatrix
	// OccRLBWT uses a run-length encoded BWT (RLBWT) for rank queries.  The
	// BWT is stored as a compact sequence of (character, length) run pairs,
	// the foundation of r-index style compressed indexes.  Rank queries run
	// in O(log r) time where r is the number of BWT runs. This is the default.
	// Indexes built
	// with this option use the FMIDX08 on-disk format.
	OccRLBWT
)

// occStructure is the internal interface for occurrence-array implementations.
type occStructure interface {
	rank(b byte, i int) int
}

// buildOcc constructs the appropriate occStructure for the given BWT.
func buildOcc(bwt []byte, typ OccStructure) occStructure {
	switch typ {
	case OccWaveletTree:
		return &waveletOcc{tree: wavelet.Build(bwt)}
	case OccWaveletMatrix:
		return &waveletMatrixOcc{mat: waveletmatrix.Build(bwt)}
	case OccRLBWT:
		return &rlbwtOcc{rl: rindex.Build(bwt)}
	default:
		return buildBitvecOcc(bwt)
	}
}

// ── bitvecOcc ─────────────────────────────────────────────────────────────────

// bitvecOcc implements occStructure using one bit-vector per character.
type bitvecOcc struct {
	vecs [256]*bitvector.BitVector
}

func buildBitvecOcc(bwt []byte) *bitvecOcc {
	n := len(bwt)
	occ := &bitvecOcc{}
	for i, b := range bwt {
		if occ.vecs[b] == nil {
			occ.vecs[b] = bitvector.New(n)
		}
		occ.vecs[b].Set(i)
	}
	for i := range occ.vecs {
		if occ.vecs[i] != nil {
			occ.vecs[i].Build()
		}
	}
	return occ
}

func (o *bitvecOcc) rank(b byte, i int) int {
	if o.vecs[b] == nil {
		return 0
	}
	return o.vecs[b].Rank1(i)
}

// ── waveletOcc ────────────────────────────────────────────────────────────────

// waveletOcc implements occStructure using a Wavelet Tree.
type waveletOcc struct {
	tree *wavelet.Tree
}

func (o *waveletOcc) rank(b byte, i int) int {
	return o.tree.Rank(b, i)
}

// ── waveletMatrixOcc ──────────────────────────────────────────────────────────

// waveletMatrixOcc implements occStructure using a Wavelet Matrix.
type waveletMatrixOcc struct {
	mat *waveletmatrix.Matrix
}

func (o *waveletMatrixOcc) rank(b byte, i int) int {
	return o.mat.Rank(b, i)
}

// ── rlbwtOcc ──────────────────────────────────────────────────────────────────

// rlbwtOcc implements occStructure using a run-length encoded BWT (RLBWT).
type rlbwtOcc struct {
	rl *rindex.RLBWT
}

func (o *rlbwtOcc) rank(b byte, i int) int {
	return o.rl.Rank(b, i)
}

// numRuns returns the number of BWT runs stored in the RLBWT.
func (o *rlbwtOcc) numRuns() int {
	return o.rl.NumRuns()
}
