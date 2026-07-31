package starfree

import (
	"errors"
	"sort"
	"testing"

	"github.com/bgnori/bwt-wheelerlang-study/internal/fmindex"
)

// --- Check tests -----------------------------------------------------------

func TestCheckAccepted(t *testing.T) {
	accepted := []string{
		"hello",
		"hello|world",
		"hel.o", // . is a character wildcard — star-free
		"[abc]at",
		"a{3}",    // bounded repeat — star-free (finite concatenation)
		"a{2,5}",  // bounded repeat — star-free (finite union of concatenations)
		"colour?", // optional — star-free (a? = a|ε)
		"(foo|bar)baz",
		"ab?c",
	}
	for _, pat := range accepted {
		if err := Check(pat); err != nil {
			t.Errorf("Check(%q) returned error, want nil: %v", pat, err)
		}
	}
}

func TestCheckRejectsUnsupportedAnchors(t *testing.T) {
	rejected := []string{
		"^hello",
		"hello$",
		"\\bword",
		"word\\B",
	}
	for _, pat := range rejected {
		err := Check(pat)
		if err == nil {
			t.Errorf("Check(%q) returned nil, want unsupported error", pat)
			continue
		}
		var ue *UnsupportedError
		if !errors.As(err, &ue) {
			t.Errorf("Check(%q) error type = %T, want *UnsupportedError", pat, err)
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

func TestSearchRejectsUnsupportedAnchors(t *testing.T) {
	idx := buildIdx("hello\nworld\nhello")
	_, err := Search(idx, "^hello", 0)
	if err == nil {
		t.Fatal("Search with anchor should return error")
	}
	var ue *UnsupportedError
	if !errors.As(err, &ue) {
		t.Fatalf("error type = %T, want *UnsupportedError", err)
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

// TestSearchJapanese verifies that star-free literal search works correctly on
// UTF-8 Japanese text.  "上杉謙信" (Uesugi Kenshin) appears twice in the
// sample sentence; the search result must report TotalCount == 2 and return
// both byte-offset positions (15 and 63).
func TestSearchJapanese(t *testing.T) {
	// 上杉謙信 appears at byte offsets 15 and 63 (each kanji = 3 bytes).
	idx := buildIdx("武田信玄と上杉謙信は戦国時代の名将である。上杉謙信は越後の虎と呼ばれた。")

	res, err := Search(idx, "上杉謙信", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", res.TotalCount)
	}

	positions := sortedInts(res.Positions(idx))
	want := []int{15, 63}
	if !intSliceEq(positions, want) {
		t.Errorf("positions = %v, want %v", positions, want)
	}

	// A name absent from the text must return zero matches.
	res, err = Search(idx, "徳川家康", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalCount != 0 {
		t.Errorf("TotalCount for absent pattern = %d, want 0", res.TotalCount)
	}
}

func TestSearchNonASCIILiteral(t *testing.T) {
	// Verify that UTF-8 (non-ASCII) literal search works correctly.
	text := "上杉謙信は戦国時代の武将である。上杉謙信の生涯は波乱万丈であった。"
	idx := buildIdx(text)

	res, err := Search(idx, "上杉謙信", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2 for '上杉謙信'", res.TotalCount)
	}
}

func TestSearchNonASCIINotFound(t *testing.T) {
	text := "上杉謙信は戦国時代の武将である。"
	idx := buildIdx(text)

	res, err := Search(idx, "武田信玄", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0 for absent Japanese pattern", res.TotalCount)
	}
}

func TestSearchMixedASCIIAndNonASCII(t *testing.T) {
	// Pattern that mixes ASCII and non-ASCII characters.
	text := "author:上杉謙信 born:1530"
	idx := buildIdx(text)

	res, err := Search(idx, "author:上杉謙信", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1 for mixed ASCII/non-ASCII pattern", res.TotalCount)
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
