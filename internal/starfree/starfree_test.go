package starfree

import (
	"sort"
	"testing"

	"github.com/bgnori/bwt-wheelerlang-study/internal/fmindex"
)

// --- Check tests -----------------------------------------------------------

func TestCheckAccepted(t *testing.T) {
	accepted := []string{
		"hello",
		"hello|world",
		"hel.o",   // . is a character wildcard — star-free
		"[abc]at",
		"a{3}",    // bounded repeat — star-free (finite concatenation)
		"a{2,5}",  // bounded repeat — star-free (finite union of concatenations)
		"colour?", // optional — star-free (a? = a|ε)
		"(foo|bar)baz",
		"ab?c",
		"^hello$", // anchors — star-free (position assertions)
	}
	for _, pat := range accepted {
		if err := Check(pat); err != nil {
			t.Errorf("Check(%q) returned error, want nil: %v", pat, err)
		}
	}
}

func TestCheckRejected(t *testing.T) {
	rejected := []string{
		"a*",
		"a+",
		"(ab)+",
		"a{2,}",
		".*",
		"a*b",
		"(a|b)*",
	}
	for _, pat := range rejected {
		if err := Check(pat); err == nil {
			t.Errorf("Check(%q) returned nil, want violation error", pat)
		}
	}
}

func TestCheckInvalidSyntax(t *testing.T) {
	if err := Check("[invalid"); err == nil {
		t.Error("Check of invalid regex should return an error")
	}
}

func TestViolationErrorMessage(t *testing.T) {
	err := Check("a*b")
	if err == nil {
		t.Fatal("expected error for a*b")
	}
	msg := err.Error()
	if msg == "" {
		t.Error("error message should not be empty")
	}
	// Should mention star or Kleene
	t.Log("violation message:", msg)
}

// --- Search tests ----------------------------------------------------------

func buildIdx(text string) *fmindex.Index {
	return fmindex.Build([]byte(text))
}

func sortedInts(xs []int) []int {
	out := append([]int(nil), xs...)
	sort.Ints(out)
	return out
}

func TestSearchLiteral(t *testing.T) {
	idx := buildIdx("abracadabra")

	res, err := Search(idx, "abra", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", res.TotalCount)
	}
	positions := sortedInts(res.Positions(idx))
	want := []int{0, 7}
	if !intSliceEq(positions, want) {
		t.Errorf("positions = %v, want %v", positions, want)
	}
}

func TestSearchAlternation(t *testing.T) {
	idx := buildIdx("the quick brown fox jumps over the lazy dog")

	res, err := Search(idx, "the|fox", 0)
	if err != nil {
		t.Fatal(err)
	}
	// "the" appears twice, "fox" once → total 3
	if res.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", res.TotalCount)
	}
}

func TestSearchDot(t *testing.T) {
	idx := buildIdx("abcde")

	res, err := Search(idx, "b.d", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalCount != 1 {
		t.Errorf("TotalCount for 'b.d' = %d, want 1", res.TotalCount)
	}
}

func TestSearchCharClass(t *testing.T) {
	idx := buildIdx("hello world")

	res, err := Search(idx, "[aeiou]", 0)
	if err != nil {
		t.Fatal(err)
	}
	// vowels in "hello world": e, o, o = 3
	if res.TotalCount != 3 {
		t.Errorf("TotalCount for vowels = %d, want 3", res.TotalCount)
	}
}

func TestSearchOptional(t *testing.T) {
	idx := buildIdx("colour color")

	// "colou?r" should match both "colour" and "color"
	res, err := Search(idx, "colou?r", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalCount < 2 {
		t.Errorf("TotalCount for 'colou?r' = %d, want >= 2", res.TotalCount)
	}
}

func TestSearchWithLimit(t *testing.T) {
	idx := buildIdx("aaaaaaaaaa") // 10 a's

	res, err := Search(idx, "a", 3)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalCount != 10 {
		t.Errorf("TotalCount = %d, want 10", res.TotalCount)
	}
	if !res.Truncated {
		t.Error("Truncated should be true when limit < total")
	}
	positions := res.Positions(idx)
	if len(positions) != 3 {
		t.Errorf("len(positions) = %d, want 3", len(positions))
	}
}

func TestSearchRejectsKleeneStar(t *testing.T) {
	idx := buildIdx("hello world")
	_, err := Search(idx, "hel*o", 0)
	if err == nil {
		t.Error("Search with Kleene star should return error")
	}
}

func TestSearchRejectsPlus(t *testing.T) {
	idx := buildIdx("hello world")
	_, err := Search(idx, "hel+o", 0)
	if err == nil {
		t.Error("Search with + should return error")
	}
}

func TestSearchRejectsUnboundedRepeat(t *testing.T) {
	idx := buildIdx("hello world")
	_, err := Search(idx, "a{2,}", 0)
	if err == nil {
		t.Error("Search with {n,} should return error")
	}
}

func TestSearchBoundedRepeat(t *testing.T) {
	idx := buildIdx("aaa bbb aaaa")
	res, err := Search(idx, "a{3}", 0)
	if err != nil {
		t.Fatal(err)
	}
	// "aaa" appears in "aaa" at pos 0, and in "aaaa" at pos 8 and 9 → 3 times
	if res.TotalCount == 0 {
		t.Error("TotalCount should be > 0 for a{3} in 'aaa bbb aaaa'")
	}
}

func TestSearchNotFound(t *testing.T) {
	idx := buildIdx("hello world")
	res, err := Search(idx, "xyz", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0 for absent pattern", res.TotalCount)
	}
}

func intSliceEq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
