// Package starfree provides validation and FM-index search for star-free
// regular expressions.
//
// # Star-free languages
//
// A language is star-free if it can be expressed by a regular expression that
// uses only:
//   - Literal characters  (a, b, …)
//   - Concatenation        (ab)
//   - Union / alternation  (a|b)
//   - Complement           (¬a)  [handled by the aperiodicity equivalence]
//   - Optional             (a?)  — syntactic sugar for a|ε
//   - Bounded repetition   (a{n,m}) — finite union of concatenations
//
// and that does NOT use:
//   - Kleene star          (a*)  — unbounded zero-or-more
//   - One-or-more          (a+)  — unbounded one-or-more
//   - Unbounded repetition (a{n,}) — equivalent to Kleene star
//
// Star-free languages are exactly the aperiodic regular languages (those
// recognised by counter-free automata).
//
// # Search
//
// Search translates the star-free pattern into a set of SA intervals on the
// FM-index using a recursive backward-search strategy: sub-expressions are
// evaluated right-to-left, accumulating a (potentially merged) list of
// non-overlapping intervals in the suffix array.
package starfree

import (
	"fmt"
	"regexp/syntax"
	"sort"

	"github.com/bgnori/bwt-wheelerlang-study/internal/fmindex"
)

// ViolationError describes a star-free constraint violation within a pattern.
type ViolationError struct {
	Op      string // human-readable name of the offending operator
	SubExpr string // string form of the offending sub-expression
}

func (e *ViolationError) Error() string {
	return fmt.Sprintf(
		"star-free violation: %q uses %s, which introduces unbounded iteration",
		e.SubExpr, e.Op,
	)
}

// Check returns nil when pattern is a valid star-free regular expression.
// It returns a *ViolationError if the pattern contains Kleene star, one-or-more,
// or an unbounded repetition; or a wrapped *syntax.Error for invalid syntax.
func Check(pattern string) error {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return fmt.Errorf("invalid regex: %w", err)
	}
	return checkNode(re)
}

func checkNode(re *syntax.Regexp) error {
	switch re.Op {
	case syntax.OpStar:
		return &ViolationError{Op: "Kleene star (*)", SubExpr: re.String()}
	case syntax.OpPlus:
		return &ViolationError{Op: "one-or-more (+)", SubExpr: re.String()}
	case syntax.OpRepeat:
		if re.Max == -1 {
			return &ViolationError{
				Op:      fmt.Sprintf("unbounded repetition {%d,}", re.Min),
				SubExpr: re.String(),
			}
		}
		// Bounded {n,m}: finite union of concatenations → star-free.
	}
	for _, sub := range re.Sub {
		if err := checkNode(sub); err != nil {
			return err
		}
	}
	return nil
}

// Interval is a half-open SA range [Lo, Hi).
type Interval struct {
	Lo, Hi int
}

// Size returns the number of matches represented by this interval.
func (iv Interval) Size() int {
	if iv.Hi > iv.Lo {
		return iv.Hi - iv.Lo
	}
	return 0
}

// SearchResult holds the outcome of a search.
type SearchResult struct {
	// Intervals contains the (merged, sorted) SA intervals matching the pattern.
	// All positions in these intervals are valid match start positions.
	Intervals []Interval
	// TotalCount is the total number of matches before any limit was applied.
	TotalCount int
	// Truncated is true when TotalCount > the requested limit.
	Truncated bool
}

// Positions returns the text positions for all intervals in the result
// using the provided index.
func (sr *SearchResult) Positions(idx *fmindex.Index) []int {
	var out []int
	for _, iv := range sr.Intervals {
		for i := iv.Lo; i < iv.Hi; i++ {
			out = append(out, idx.SAAt(i))
		}
	}
	return out
}

// Search executes a star-free regex search on the FM-index.
//
// pattern must be a valid star-free regular expression (see Check).
// limit controls the maximum number of match positions returned; ≤0 means no limit.
//
// Returns a *ViolationError if pattern violates star-free constraints, or a
// wrapped *syntax.Error for invalid syntax.
func Search(idx *fmindex.Index, pattern string, limit int) (*SearchResult, error) {
	if err := Check(pattern); err != nil {
		return nil, err
	}
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}
	re = re.Simplify()

	ivs := evalRegex(idx, re, 0, idx.SALen())

	total := 0
	for _, iv := range ivs {
		total += iv.Size()
	}

	truncated := false
	if limit > 0 && total > limit {
		truncated = true
		ivs = truncateIntervals(ivs, limit)
	}

	return &SearchResult{
		Intervals:  ivs,
		TotalCount: total,
		Truncated:  truncated,
	}, nil
}

// evalRegex recursively evaluates re against the FM-index within SA interval
// [lo, hi) and returns the resulting set of (non-overlapping, sorted) intervals.
func evalRegex(idx *fmindex.Index, re *syntax.Regexp, lo, hi int) []Interval {
	switch re.Op {
	case syntax.OpLiteral:
		return evalLiteral(idx, re.Rune, lo, hi, re.Flags&syntax.FoldCase != 0)

	case syntax.OpCharClass:
		return evalCharClass(idx, re.Rune, lo, hi)

	case syntax.OpAnyCharNotNL:
		return evalAnyChar(idx, lo, hi, false)

	case syntax.OpAnyChar:
		return evalAnyChar(idx, lo, hi, true)

	case syntax.OpConcat:
		// Process sub-expressions right-to-left (backward search order).
		if len(re.Sub) == 0 {
			return []Interval{{lo, hi}}
		}
		intervals := []Interval{{lo, hi}}
		for i := len(re.Sub) - 1; i >= 0 && len(intervals) > 0; i-- {
			var next []Interval
			for _, iv := range intervals {
				next = append(next, evalRegex(idx, re.Sub[i], iv.Lo, iv.Hi)...)
			}
			intervals = mergeIntervals(next)
		}
		return intervals

	case syntax.OpAlternate:
		var result []Interval
		for _, sub := range re.Sub {
			result = append(result, evalRegex(idx, sub, lo, hi)...)
		}
		return mergeIntervals(result)

	case syntax.OpQuest:
		// a? = a | ε  — either apply sub-expression or leave interval unchanged.
		result := evalRegex(idx, re.Sub[0], lo, hi)
		result = append(result, Interval{lo, hi})
		return mergeIntervals(result)

	case syntax.OpRepeat:
		// Bounded {n,m}: already verified max != -1 by Check.
		return evalBoundedRepeat(idx, re.Sub[0], re.Min, re.Max, lo, hi)

	case syntax.OpCapture:
		if len(re.Sub) > 0 {
			return evalRegex(idx, re.Sub[0], lo, hi)
		}
		return []Interval{{lo, hi}}

	case syntax.OpEmptyMatch:
		return []Interval{{lo, hi}}

	// Position anchors (^, $, \b, \B) cannot be expressed in the FM-index
	// backward-search model; treat them as empty matches.
	case syntax.OpBeginText, syntax.OpEndText,
		syntax.OpBeginLine, syntax.OpEndLine,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return []Interval{{lo, hi}}
	}

	return nil
}

// evalLiteral processes a multi-character literal (re.Rune) right-to-left.
// foldCase enables case-insensitive matching for ASCII letters.
// Non-ASCII runes are encoded as their UTF-8 byte sequences and each byte is
// fed into the backward search in right-to-left order.
func evalLiteral(idx *fmindex.Index, runes []rune, lo, hi int, foldCase bool) []Interval {
	intervals := []Interval{{lo, hi}}
	for i := len(runes) - 1; i >= 0 && len(intervals) > 0; i-- {
		r := runes[i]

		// Build the set of byte sequences to try for this rune.
		// ASCII: optionally case-fold to a second sequence.
		// Non-ASCII: one UTF-8 byte sequence; case-folding is not applied.
		var seqs [][]byte
		if r <= 127 {
			b := byte(r)
			if foldCase {
				switch {
				case b >= 'a' && b <= 'z':
					seqs = [][]byte{{b}, {b - 32}}
				case b >= 'A' && b <= 'Z':
					seqs = [][]byte{{b}, {b + 32}}
				default:
					seqs = [][]byte{{b}}
				}
			} else {
				seqs = [][]byte{{b}}
			}
		} else {
			// Encode the rune as UTF-8 and treat each byte individually.
			seqs = [][]byte{[]byte(string(r))}
		}

		var next []Interval
		for _, iv := range intervals {
			for _, bs := range seqs {
				// Apply backward search steps right-to-left over the bytes of
				// this rune (or character).
				curLo, curHi := iv.Lo, iv.Hi
				for j := len(bs) - 1; j >= 0 && curLo < curHi; j-- {
					curLo, curHi = idx.BackwardSearchStep(bs[j], curLo, curHi)
				}
				if curLo < curHi {
					next = append(next, Interval{curLo, curHi})
				}
			}
		}
		intervals = mergeIntervals(next)
	}
	return intervals
}

// evalCharClass handles a character class (re.Rune = [lo1,hi1, lo2,hi2, …]).
func evalCharClass(idx *fmindex.Index, runes []rune, lo, hi int) []Interval {
	var result []Interval
	for k := 0; k+1 < len(runes); k += 2 {
		for r := runes[k]; r <= runes[k+1] && r <= 127; r++ {
			b := byte(r)
			newLo, newHi := idx.BackwardSearchStep(b, lo, hi)
			if newLo < newHi {
				result = append(result, Interval{newLo, newHi})
			}
		}
	}
	return mergeIntervals(result)
}

// evalAnyChar handles the dot (.) metacharacter.
func evalAnyChar(idx *fmindex.Index, lo, hi int, includeNewline bool) []Interval {
	var result []Interval
	for c := 1; c <= 127; c++ { // skip null (sentinel)
		if !includeNewline && c == '\n' {
			continue
		}
		b := byte(c)
		newLo, newHi := idx.BackwardSearchStep(b, lo, hi)
		if newLo < newHi {
			result = append(result, Interval{newLo, newHi})
		}
	}
	return mergeIntervals(result)
}

// evalBoundedRepeat handles {min,max} bounded repetition by expansion.
func evalBoundedRepeat(idx *fmindex.Index, sub *syntax.Regexp, min, max int, lo, hi int) []Interval {
	// Apply sub exactly `min` times first.
	intervals := []Interval{{lo, hi}}
	for i := 0; i < min && len(intervals) > 0; i++ {
		var next []Interval
		for _, iv := range intervals {
			next = append(next, evalRegex(idx, sub, iv.Lo, iv.Hi)...)
		}
		intervals = mergeIntervals(next)
	}

	if min == max {
		return intervals
	}

	// Collect results for min, min+1, …, max repetitions.
	all := append([]Interval(nil), intervals...)
	for i := min; i < max && len(intervals) > 0; i++ {
		var next []Interval
		for _, iv := range intervals {
			next = append(next, evalRegex(idx, sub, iv.Lo, iv.Hi)...)
		}
		intervals = mergeIntervals(next)
		all = append(all, intervals...)
	}
	return mergeIntervals(all)
}

// --- Interval helpers -------------------------------------------------------

// mergeIntervals sorts intervals by Lo and merges overlapping/adjacent ones.
func mergeIntervals(ivs []Interval) []Interval {
	if len(ivs) == 0 {
		return nil
	}
	sort.Slice(ivs, func(i, j int) bool { return ivs[i].Lo < ivs[j].Lo })
	result := ivs[:1:1]
	for _, iv := range ivs[1:] {
		last := &result[len(result)-1]
		if iv.Lo <= last.Hi {
			if iv.Hi > last.Hi {
				last.Hi = iv.Hi
			}
		} else {
			result = append(result, iv)
		}
	}
	return result
}

// truncateIntervals reduces intervals to contain at most n total positions.
func truncateIntervals(ivs []Interval, n int) []Interval {
	var out []Interval
	remaining := n
	for _, iv := range ivs {
		if remaining <= 0 {
			break
		}
		size := iv.Size()
		if size <= remaining {
			out = append(out, iv)
			remaining -= size
		} else {
			out = append(out, Interval{iv.Lo, iv.Lo + remaining})
			remaining = 0
		}
	}
	return out
}
