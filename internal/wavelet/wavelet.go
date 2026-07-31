// Package wavelet provides a Wavelet Tree for rank queries over byte sequences.
//
// A Wavelet Tree partitions the alphabet [0, 256) recursively in halves,
// storing a succinct bit-vector at each internal node that records which
// elements belong to the upper half of the current alphabet range.  Rank
// queries run in O(log 256) = O(8) time regardless of the alphabet size.
package wavelet

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/bgnori/bwt-wheelerlang-study/internal/bitvector"
)

// Tree is a Wavelet Tree built over a byte sequence.
type Tree struct {
	n    int
	root *node
}

// node is an internal node of the Wavelet Tree.
// Leaves (where hi-lo == 1) are represented as nil because no bit-vector is
// needed: all elements at a leaf are the same character.
type node struct {
	bv    *bitvector.BitVector
	left  *node
	right *node
}

// Build constructs a Wavelet Tree from the byte sequence seq.
func Build(seq []byte) *Tree {
	t := &Tree{n: len(seq)}
	if len(seq) > 0 {
		t.root = buildNode(seq, 0, 256)
	}
	return t
}

// buildNode recursively builds a wavelet tree node covering alphabet [lo, hi).
// Returns nil for leaf nodes (hi-lo <= 1) or empty sequences.
func buildNode(seq []byte, lo, hi int) *node {
	if len(seq) == 0 || hi-lo <= 1 {
		return nil
	}
	mid := (lo + hi) / 2
	nd := &node{bv: bitvector.New(len(seq))}

	var leftSeq, rightSeq []byte
	for i, c := range seq {
		if int(c) >= mid {
			nd.bv.Set(i)
			rightSeq = append(rightSeq, c)
		} else {
			leftSeq = append(leftSeq, c)
		}
	}
	nd.bv.Build()
	nd.left = buildNode(leftSeq, lo, mid)
	nd.right = buildNode(rightSeq, mid, hi)
	return nd
}

// Rank returns the number of occurrences of c in seq[0..i).
// i is clamped to [0, n].
func (t *Tree) Rank(c byte, i int) int {
	if t.root == nil || i <= 0 {
		return 0
	}
	if i > t.n {
		i = t.n
	}
	return rankNode(t.root, int(c), i, 0, 256)
}

// rankNode navigates the tree to answer Rank(c, i) within alphabet [lo, hi).
func rankNode(nd *node, c, i, lo, hi int) int {
	if i <= 0 {
		return 0
	}
	// Leaf: all elements in this subtree have value lo.
	if hi-lo <= 1 {
		if c == lo {
			return i
		}
		return 0
	}
	// Empty subtree (no character in [lo, hi) appeared in the sequence).
	if nd == nil {
		return 0
	}
	mid := (lo + hi) / 2
	if c < mid {
		// Navigate left: count 0-bits (characters < mid) in [0, i).
		return rankNode(nd.left, c, nd.bv.Rank0(i), lo, mid)
	}
	// Navigate right: count 1-bits (characters >= mid) in [0, i).
	return rankNode(nd.right, c, nd.bv.Rank1(i), mid, hi)
}

// WriteTo serialises the Wavelet Tree to w in a compact binary format.
// The tree is written in DFS pre-order.  It implements io.WriterTo.
func (t *Tree) WriteTo(w io.Writer) (int64, error) {
	var written int64
	if err := binary.Write(w, binary.LittleEndian, int64(t.n)); err != nil {
		return 0, fmt.Errorf("wavelet: write n: %w", err)
	}
	written += 8
	n, err := writeNode(w, t.root)
	written += n
	return written, err
}

// writeNode writes a single node (and its subtree) to w.
// A one-byte presence flag precedes each node: 0 = nil, 1 = present.
func writeNode(w io.Writer, nd *node) (int64, error) {
	var written int64
	if nd == nil {
		if err := writeByte(w, 0); err != nil {
			return 0, fmt.Errorf("wavelet: write node absent: %w", err)
		}
		return 1, nil
	}
	if err := writeByte(w, 1); err != nil {
		return 0, fmt.Errorf("wavelet: write node present: %w", err)
	}
	written++

	n, err := nd.bv.WriteTo(w)
	written += n
	if err != nil {
		return written, fmt.Errorf("wavelet: write bv: %w", err)
	}
	n, err = writeNode(w, nd.left)
	written += n
	if err != nil {
		return written, err
	}
	n, err = writeNode(w, nd.right)
	written += n
	return written, err
}

// ReadFrom deserialises a Wavelet Tree from r.
func ReadFrom(r io.Reader) (*Tree, error) {
	var n64 int64
	if err := binary.Read(r, binary.LittleEndian, &n64); err != nil {
		return nil, fmt.Errorf("wavelet: read n: %w", err)
	}
	root, err := readNode(r)
	if err != nil {
		return nil, fmt.Errorf("wavelet: read tree: %w", err)
	}
	return &Tree{n: int(n64), root: root}, nil
}

// readNode reads one node (and its subtree) from r.
func readNode(r io.Reader) (*node, error) {
	presence, err := readByte(r)
	if err != nil {
		return nil, fmt.Errorf("wavelet: read node flag: %w", err)
	}
	if presence == 0 {
		return nil, nil
	}
	bv, err := bitvector.ReadFrom(r)
	if err != nil {
		return nil, fmt.Errorf("wavelet: read bv: %w", err)
	}
	left, err := readNode(r)
	if err != nil {
		return nil, err
	}
	right, err := readNode(r)
	if err != nil {
		return nil, err
	}
	return &node{bv: bv, left: left, right: right}, nil
}

// writeByte writes a single byte to w.
func writeByte(w io.Writer, b byte) error {
	_, err := w.Write([]byte{b})
	return err
}

// readByte reads a single byte from r.
func readByte(r io.Reader) (byte, error) {
	buf := [1]byte{}
	_, err := io.ReadFull(r, buf[:])
	return buf[0], err
}
