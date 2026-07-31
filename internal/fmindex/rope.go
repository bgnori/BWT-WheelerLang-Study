package fmindex

// rope is a balanced binary tree of byte chunks.
type rope struct {
	left   *rope
	right  *rope
	chunk  []byte
	weight int
	height int
}

func newRopeFromBytes(b []byte) *rope {
	if len(b) == 0 {
		return nil
	}
	return newLeaf(b)
}

func newLeaf(b []byte) *rope {
	cp := append([]byte(nil), b...)
	return &rope{
		chunk:  cp,
		weight: len(cp),
		height: 1,
	}
}

func (r *rope) Append(b []byte) *rope {
	if len(b) == 0 {
		return r
	}
	return concatRope(r, newLeaf(b))
}

func concatRope(a, b *rope) *rope {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return rebalance(&rope{
		left:   a,
		right:  b,
		weight: lenRope(a) + lenRope(b),
		height: max(height(a), height(b)) + 1,
	})
}

func (r *rope) Bytes() []byte {
	if r == nil {
		return nil
	}
	out := make([]byte, 0, lenRope(r))
	r.collect(&out)
	return out
}

func (r *rope) collect(out *[]byte) {
	if r == nil {
		return
	}
	if r.left == nil && r.right == nil {
		*out = append(*out, r.chunk...)
		return
	}
	r.left.collect(out)
	r.right.collect(out)
}

func height(r *rope) int {
	if r == nil {
		return 0
	}
	return r.height
}

func lenRope(r *rope) int {
	if r == nil {
		return 0
	}
	return r.weight
}

func update(r *rope) {
	if r == nil {
		return
	}
	if r.left == nil && r.right == nil {
		r.weight = len(r.chunk)
		r.height = 1
		return
	}
	r.weight = lenRope(r.left) + lenRope(r.right)
	r.height = max(height(r.left), height(r.right)) + 1
}

func balance(r *rope) int {
	if r == nil {
		return 0
	}
	return height(r.left) - height(r.right)
}

func rebalance(r *rope) *rope {
	update(r)
	bf := balance(r)
	if bf > 1 {
		if balance(r.left) < 0 {
			r.left = rotateLeft(r.left)
		}
		return rotateRight(r)
	}
	if bf < -1 {
		if balance(r.right) > 0 {
			r.right = rotateRight(r.right)
		}
		return rotateLeft(r)
	}
	return r
}

func rotateLeft(x *rope) *rope {
	y := x.right
	t2 := y.left
	y.left = x
	x.right = t2
	update(x)
	update(y)
	return y
}

func rotateRight(y *rope) *rope {
	x := y.left
	t2 := x.right
	x.right = y
	y.left = t2
	update(y)
	update(x)
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
