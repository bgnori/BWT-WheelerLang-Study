package bwtsearch

// Tests for building an FM-index over multiple source files, mirroring the
// Moby Dick and Kenshin single-file workflows but adapted for a multi-file
// corpus such as the Git source code.
//
// Three groups of tests are provided:
//
//  1. Unit tests for BuildFromFiles that always run and need no external data.
//  2. Integration tests using the repository's own Go source files (always
//     available in CI).
//  3. Integration tests using the downloaded Git source tree; these are skipped
//     unless data/git-src/ has been populated by scripts/download_git.sh.

import (
	"bytes"
	"index/suffixarray"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// ─── helpers ────────────────────────────────────────────────────────────────

// collectSourceFiles walks root and returns the contents of every file whose
// extension is in exts (e.g. ".go", ".c", ".h"), together with the
// corresponding file names.
func collectSourceFiles(root string, exts map[string]bool) (texts [][]byte, names []string, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}
		if info.IsDir() {
			return nil
		}
		if exts[filepath.Ext(info.Name())] {
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			texts = append(texts, data)
			names = append(names, info.Name())
		}
		return nil
	})
	return
}

// sortedIntSlice returns a sorted copy of xs.
func sortedIntSlice(xs []int) []int {
	out := append([]int(nil), xs...)
	sort.Ints(out)
	return out
}

// verifyCountsMatchStdlib builds a stdlib suffix array over combined (the same
// byte slice used to build idx) and checks that FM-index Count equals stdlib
// Lookup for each pattern.
func verifyCountsMatchStdlib(t *testing.T, idx *Index, combined []byte, patterns []string) {
	t.Helper()
	sa := suffixarray.New(combined)
	for _, pat := range patterns {
		p := []byte(pat)
		fmCount := idx.Count(p)
		stdCount := len(sa.Lookup(p, -1))
		if fmCount != stdCount {
			t.Errorf("Count(%q): FM-index=%d stdlib=%d", pat, fmCount, stdCount)
		}
	}
}

// ─── Unit tests for BuildFromFiles ──────────────────────────────────────────

// TestBuildFromFilesBasic verifies that BuildFromFiles correctly merges
// multiple byte slices and counts occurrences that span across them.
func TestBuildFromFilesBasic(t *testing.T) {
	files := [][]byte{
		[]byte("hello world"),
		[]byte("world peace"),
		[]byte("hello again"),
	}
	sep := []byte("\n")
	idx := BuildFromFiles(files, sep)
	combined := bytes.Join(files, sep)

	cases := []struct {
		pat  string
		want int
	}{
		{"hello", 2},
		{"world", 2},
		{"peace", 1},
		{"again", 1},
		{"xyz", 0},
	}
	for _, tc := range cases {
		if got := idx.Count([]byte(tc.pat)); got != tc.want {
			t.Errorf("Count(%q) = %d, want %d", tc.pat, got, tc.want)
		}
	}
	verifyCountsMatchStdlib(t, idx, combined, []string{"hello", "world", "peace", "again", "xyz"})
}

// TestBuildFromFilesNilSeparator checks that a nil separator defaults to "\n",
// producing the same index as an explicit newline separator.
func TestBuildFromFilesNilSeparator(t *testing.T) {
	files := [][]byte{
		[]byte("foo bar"),
		[]byte("baz qux"),
	}
	idxNil := BuildFromFiles(files, nil)
	idxNL := BuildFromFiles(files, []byte("\n"))
	if idxNil.TextLen() != idxNL.TextLen() {
		t.Errorf("TextLen mismatch: nil sep=%d, newline sep=%d", idxNil.TextLen(), idxNL.TextLen())
	}
	for _, pat := range []string{"foo", "bar", "baz", "qux"} {
		if idxNil.Count([]byte(pat)) != idxNL.Count([]byte(pat)) {
			t.Errorf("Count(%q) differs between nil and newline separator", pat)
		}
	}
}

// TestBuildFromFilesEmptyList checks that BuildFromFiles handles an empty
// input slice gracefully.
func TestBuildFromFilesEmptyList(t *testing.T) {
	idx := BuildFromFiles(nil, nil)
	if idx == nil {
		t.Fatal("BuildFromFiles(nil, nil) returned nil")
	}
	if got := idx.Count([]byte("x")); got != 0 {
		t.Errorf("empty index: Count(x) = %d, want 0", got)
	}
}

// TestBuildFromFilesSingleFile checks that a one-element list is equivalent
// to calling Build directly.
func TestBuildFromFilesSingleFile(t *testing.T) {
	text := []byte("abracadabra")
	idx1 := Build(text)
	idx2 := BuildFromFiles([][]byte{text}, nil)
	for _, pat := range []string{"abra", "bra", "cadabra", "xyz"} {
		c1 := idx1.Count([]byte(pat))
		c2 := idx2.Count([]byte(pat))
		if c1 != c2 {
			t.Errorf("Count(%q): Build=%d BuildFromFiles=%d", pat, c1, c2)
		}
	}
}

// ─── Integration tests: repository's own Go source files ────────────────────

// TestGoSourceMultiFileIndex builds an FM-index from the Go source files in
// the repository root and verifies that common code patterns (func, package,
// import, error, return) are found and match the stdlib suffix array.
//
// This test always runs and requires no external downloads; it models the same
// workflow as the Moby Dick / Kenshin single-file tests but over a multi-file
// Go source corpus.
func TestGoSourceMultiFileIndex(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	texts, names, err := collectSourceFiles(dir, map[string]bool{".go": true})
	if err != nil {
		t.Fatalf("collectSourceFiles: %v", err)
	}
	if len(texts) == 0 {
		t.Skip("no .go files found in working directory")
	}
	t.Logf("indexing %d Go source files: %v", len(texts), names)

	sep := []byte("\n")
	idx := BuildFromFiles(texts, sep)
	combined := bytes.Join(texts, sep)

	// Patterns that must appear in any non-trivial Go source corpus.
	patterns := []string{"func", "package", "import", "error", "return"}
	for _, pat := range patterns {
		if got := idx.Count([]byte(pat)); got == 0 {
			t.Errorf("Count(%q) = 0; expected at least one occurrence in Go sources", pat)
		}
	}
	verifyCountsMatchStdlib(t, idx, combined, patterns)
}

// TestGoSourceMultiFilePersistence verifies that an FM-index built from
// multiple Go source files survives a save/reload round-trip.
func TestGoSourceMultiFilePersistence(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	texts, _, err := collectSourceFiles(dir, map[string]bool{".go": true})
	if err != nil {
		t.Fatalf("collectSourceFiles: %v", err)
	}
	if len(texts) == 0 {
		t.Skip("no .go files found in working directory")
	}

	idx := BuildFromFiles(texts, []byte("\n"))

	var buf bytes.Buffer
	if _, err := idx.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	loaded, err := ReadFrom(&buf)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	for _, pat := range []string{"func", "package", "import"} {
		orig := idx.Count([]byte(pat))
		reloaded := loaded.Count([]byte(pat))
		if orig != reloaded {
			t.Errorf("Count(%q) after reload: original=%d reloaded=%d", pat, orig, reloaded)
		}
	}
}

// TestGoSourceMultiFileSearch exercises star-free regex search over the
// multi-file Go source index.
func TestGoSourceMultiFileSearch(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	texts, _, err := collectSourceFiles(dir, map[string]bool{".go": true})
	if err != nil {
		t.Fatalf("collectSourceFiles: %v", err)
	}
	if len(texts) == 0 {
		t.Skip("no .go files found in working directory")
	}

	idx := BuildFromFiles(texts, []byte("\n"))

	// "func|type" should find occurrences of both keywords.
	res, err := Search(idx, "func|type", 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.TotalCount == 0 {
		t.Error("Search(func|type) returned 0 results")
	}

	// Kleene star must be rejected even over the multi-file index.
	if err := Check("func.*"); err == nil {
		t.Error("Check(func.*) should return a star-free violation error")
	}
}

// ─── Integration tests: Git source tree ─────────────────────────────────────

// gitSrcDir returns the path to the downloaded Git source directory, or ""
// if it has not been populated.
func gitSrcDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	p := filepath.Join(dir, "data", "git-src")
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

// TestGitSourceMultiFileIndex builds an FM-index from Git's C and header
// source files and verifies that common C/Git patterns (commit, diff, struct,
// #include) are found and match the stdlib suffix array.
//
// Run ./scripts/download_git.sh first to populate data/git-src/.
func TestGitSourceMultiFileIndex(t *testing.T) {
	srcDir := gitSrcDir()
	if srcDir == "" {
		t.Skip("Git source not found at data/git-src; run ./scripts/download_git.sh")
	}

	texts, names, err := collectSourceFiles(srcDir, map[string]bool{".c": true, ".h": true})
	if err != nil {
		t.Fatalf("collectSourceFiles: %v", err)
	}
	if len(texts) == 0 {
		t.Skip("no .c/.h files found in data/git-src")
	}
	t.Logf("indexing %d C/header files from Git source", len(texts))
	_ = names

	sep := []byte("\n")
	idx := BuildFromFiles(texts, sep)
	combined := bytes.Join(texts, sep)

	// Patterns typical of C source and the Git codebase specifically.
	patterns := []string{"commit", "diff", "struct", "#include"}
	for _, pat := range patterns {
		if got := idx.Count([]byte(pat)); got == 0 {
			t.Errorf("Count(%q) = 0; expected occurrences in Git source", pat)
		}
		t.Logf("Count(%q) = %d", pat, idx.Count([]byte(pat)))
	}
	verifyCountsMatchStdlib(t, idx, combined, patterns)
}

// TestGitSourceMultiFilePersistence verifies that a Git-source FM-index
// survives a save-to-file / reload round-trip.
func TestGitSourceMultiFilePersistence(t *testing.T) {
	srcDir := gitSrcDir()
	if srcDir == "" {
		t.Skip("Git source not found at data/git-src; run ./scripts/download_git.sh")
	}

	texts, _, err := collectSourceFiles(srcDir, map[string]bool{".c": true, ".h": true})
	if err != nil {
		t.Fatalf("collectSourceFiles: %v", err)
	}
	if len(texts) == 0 {
		t.Skip("no .c/.h files found in data/git-src")
	}

	idx := BuildFromFiles(texts, []byte("\n"))

	f, err := os.CreateTemp("", "git-src-*.idx")
	if err != nil {
		t.Fatal(err)
	}
	idxPath := f.Name()
	f.Close()
	defer os.Remove(idxPath)

	if err := idx.Save(idxPath); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(idxPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, pat := range []string{"commit", "diff", "#include"} {
		orig := idx.Count([]byte(pat))
		reloaded := loaded.Count([]byte(pat))
		if orig != reloaded {
			t.Errorf("Count(%q) after reload: original=%d reloaded=%d", pat, orig, reloaded)
		}
	}
}

// TestGitSourceMultiFileLocate checks that Locate results for a pattern in
// the Git source index match the stdlib suffix array positions.
func TestGitSourceMultiFileLocate(t *testing.T) {
	srcDir := gitSrcDir()
	if srcDir == "" {
		t.Skip("Git source not found at data/git-src; run ./scripts/download_git.sh")
	}

	texts, _, err := collectSourceFiles(srcDir, map[string]bool{".c": true, ".h": true})
	if err != nil {
		t.Fatalf("collectSourceFiles: %v", err)
	}
	if len(texts) == 0 {
		t.Skip("no .c/.h files found in data/git-src")
	}

	sep := []byte("\n")
	idx := BuildFromFiles(texts, sep)
	combined := bytes.Join(texts, sep)
	sa := suffixarray.New(combined)

	pat := []byte("commit")
	fmPositions := sortedIntSlice(idx.Locate(pat, 0))
	stdPositions := sortedIntSlice(sa.Lookup(pat, -1))

	if len(fmPositions) != len(stdPositions) {
		t.Fatalf("Locate(commit): FM-index=%d positions, stdlib=%d positions",
			len(fmPositions), len(stdPositions))
	}
	for i := range fmPositions {
		if fmPositions[i] != stdPositions[i] {
			t.Errorf("position[%d]: FM-index=%d stdlib=%d", i, fmPositions[i], stdPositions[i])
		}
	}
}

// TestGitSourceMultiFileSearch exercises star-free regex search over the Git
// source index.
func TestGitSourceMultiFileSearch(t *testing.T) {
	srcDir := gitSrcDir()
	if srcDir == "" {
		t.Skip("Git source not found at data/git-src; run ./scripts/download_git.sh")
	}

	texts, _, err := collectSourceFiles(srcDir, map[string]bool{".c": true, ".h": true})
	if err != nil {
		t.Fatalf("collectSourceFiles: %v", err)
	}
	if len(texts) == 0 {
		t.Skip("no .c/.h files found in data/git-src")
	}

	idx := BuildFromFiles(texts, []byte("\n"))

	// "commit|diff" — alternation of two Git keywords.
	res, err := Search(idx, "commit|diff", 0)
	if err != nil {
		t.Fatalf("Search(commit|diff): %v", err)
	}
	if res.TotalCount == 0 {
		t.Error("Search(commit|diff) returned 0 results")
	}
	t.Logf("Search(commit|diff) TotalCount = %d", res.TotalCount)

	// Kleene star must still be rejected.
	if err := Check("commit.*"); err == nil {
		t.Error("Check(commit.*) should return a star-free violation error")
	}
}
