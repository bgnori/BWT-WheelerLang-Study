package fmindex

import (
	"github.com/bgnori/bwt-wheelerlang-study/internal/bitvector"
	"github.com/bgnori/bwt-wheelerlang-study/internal/wavelet"
)

// OccStructure selects the occurrence-array implementation used inside an
// FM-index.
type OccStructure int

const (
	// OccBitvectors uses one succinct bit-vector per distinct character.
	// This is the default and matches the existing FMIDX01 on-disk format.
	OccBitvectors OccStructure = iota
	// OccWaveletTree uses a Wavelet Tree over the BWT, providing O(log σ)
	// rank queries and O(n log σ) total space.  Indexes built with this
	// option are written in the FMIDX02 on-disk format.
	OccWaveletTree
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
