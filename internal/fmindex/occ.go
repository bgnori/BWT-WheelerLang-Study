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
	// OccRRR uses one RRR bit-vector per distinct character over the BWT.
	// Indexes built with this option use the FMIDX09 on-disk format.
	OccRRR
	// OccEliasFano uses one Elias-Fano encoded position list per distinct
	// character over the BWT. Indexes built with this option use the FMIDX10
	// on-disk format.
	OccEliasFano
	// OccPoppy uses one interleaved RRR (Poppy-style) bit-vector per distinct
	// character over the BWT. Indexes built with this option use the FMIDX11
	// on-disk format.
	OccPoppy
	// OccDynamicBitvectors uses one dynamic bit-vector per distinct character
	// over the BWT. Indexes built with this option use the FMIDX12 on-disk
	// format.
	OccDynamicBitvectors
	// OccExternalWaveletTree uses an external-memory Wavelet Tree over the BWT.
	// Node bit-vectors are stored in temporary files with in-memory rank
	// summaries. This constant is kept for backward compatibility.
	// Prefer OccWaveletTree with OccStorageExternal.
	// Indexes built with this option use the FMIDX13 on-disk format.
	OccExternalWaveletTree
)

// OccStorageMode selects the physical storage strategy for occ structures.
type OccStorageMode int

const (
	// OccStorageInMemory stores occ structures fully in memory.
	OccStorageInMemory OccStorageMode = iota
	// OccStorageExternal stores supported occ structures with external memory.
	// Currently, this mode is supported for OccWaveletTree.
	OccStorageExternal
)

// OccStorageOptions controls physical storage parameters for occ structures.
type OccStorageOptions struct {
	Mode          OccStorageMode
	DiskBlockSize int
}

func defaultOccStorageOptions() OccStorageOptions {
	return OccStorageOptions{Mode: OccStorageInMemory, DiskBlockSize: 4096}
}

func normalizeOccStorageOptions(opt OccStorageOptions) OccStorageOptions {
	def := defaultOccStorageOptions()
	if opt.Mode != OccStorageExternal {
		opt.Mode = OccStorageInMemory
	}
	if opt.DiskBlockSize <= 0 {
		opt.DiskBlockSize = def.DiskBlockSize
	}
	if opt.DiskBlockSize < 8 {
		opt.DiskBlockSize = 8
	}
	if rem := opt.DiskBlockSize % 8; rem != 0 {
		opt.DiskBlockSize += 8 - rem
	}
	return opt
}

func normalizeOccConfig(typ OccStructure, opt OccStorageOptions) (OccStructure, OccStorageOptions) {
	opt = normalizeOccStorageOptions(opt)
	if typ == OccExternalWaveletTree {
		typ = OccWaveletTree
		opt.Mode = OccStorageExternal
	}
	if opt.Mode == OccStorageExternal && typ != OccWaveletTree {
		opt.Mode = OccStorageInMemory
	}
	return typ, opt
}

// occStructure is the internal interface for occurrence-array implementations.
type occStructure interface {
	rank(b byte, i int) int
}

// buildOcc constructs the appropriate occStructure for the given BWT.
func buildOccWithStorage(bwt []byte, typ OccStructure, storage OccStorageOptions) occStructure {
	typ, storage = normalizeOccConfig(typ, storage)
	switch typ {
	case OccWaveletTree:
		if storage.Mode == OccStorageExternal {
			cfg := wavelet.ExternalConfig{DiskBlockSize: storage.DiskBlockSize}
			return &externalWaveletOcc{tree: wavelet.BuildExternalWithConfig(bwt, cfg)}
		}
		return &waveletOcc{tree: wavelet.Build(bwt)}
	case OccWaveletMatrix:
		return &waveletMatrixOcc{mat: waveletmatrix.Build(bwt)}
	case OccRLBWT:
		return &rlbwtOcc{rl: rindex.Build(bwt)}
	case OccRRR:
		return buildRRROcc(bwt)
	case OccEliasFano:
		return buildEliasFanoOcc(bwt)
	case OccPoppy:
		return buildPoppyOcc(bwt)
	case OccDynamicBitvectors:
		return buildDynamicBitvecOcc(bwt)
	default:
		return buildBitvecOcc(bwt)
	}
}

// buildOcc constructs the default in-memory occ structure for the given BWT.
func buildOcc(bwt []byte, typ OccStructure) occStructure {
	return buildOccWithStorage(bwt, typ, defaultOccStorageOptions())
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

// ── dynamicBitvecOcc ────────────────────────────────────────────────────────

// dynamicBitvecOcc implements occStructure using one dynamic bit-vector per
// character.
type dynamicBitvecOcc struct {
	vecs [256]*bitvector.DynamicBitVector
}

func buildDynamicBitvecOcc(bwt []byte) *dynamicBitvecOcc {
	n := len(bwt)
	occ := &dynamicBitvecOcc{}
	for i, b := range bwt {
		if occ.vecs[b] == nil {
			occ.vecs[b] = bitvector.NewDynamic(n)
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

func (o *dynamicBitvecOcc) rank(b byte, i int) int {
	if o.vecs[b] == nil {
		return 0
	}
	return o.vecs[b].Rank1(i)
}

// ── rrrOcc ──────────────────────────────────────────────────────────────────

// rrrOcc implements occStructure using one RRR bit-vector per character.
type rrrOcc struct {
	vecs [256]*bitvector.RRRVector
}

func buildRRROcc(bwt []byte) *rrrOcc {
	n := len(bwt)
	occ := &rrrOcc{}
	for i, b := range bwt {
		if occ.vecs[b] == nil {
			occ.vecs[b] = bitvector.NewRRR(n)
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

func (o *rrrOcc) rank(b byte, i int) int {
	if o.vecs[b] == nil {
		return 0
	}
	return o.vecs[b].Rank1(i)
}

// ── poppyOcc ────────────────────────────────────────────────────────────────

// poppyOcc implements occStructure using one Poppy (interleaved RRR)
// bit-vector per character.
type poppyOcc struct {
	vecs [256]*bitvector.PoppyVector
}

func buildPoppyOcc(bwt []byte) *poppyOcc {
	n := len(bwt)
	occ := &poppyOcc{}
	for i, b := range bwt {
		if occ.vecs[b] == nil {
			occ.vecs[b] = bitvector.NewPoppy(n)
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

func (o *poppyOcc) rank(b byte, i int) int {
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

// ── externalWaveletOcc ───────────────────────────────────────────────────────

// externalWaveletOcc implements occStructure using an external-memory
// Wavelet Tree.
type externalWaveletOcc struct {
	tree *wavelet.ExternalTree
}

func (o *externalWaveletOcc) rank(b byte, i int) int {
	return o.tree.Rank(b, i)
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
