// Package bwtsearch – performance benchmarks.
//
// These benchmarks measure index construction cost and search cost across
// three corpora (Moby Dick, 上杉謙信/Kenshin, Git source) and three
// construction algorithms (Doubling, SA-IS, Go stdlib suffixarray).
//
// # Prerequisites
//
// Download the corpora before running these benchmarks:
//
//	./scripts/download_testdata.sh   # Moby Dick
//	./scripts/download_kenshin.sh    # 上杉謙信
//	./scripts/download_git.sh        # Git source (.c/.h files)
//
// # Run
//
//	go test -bench=. -benchmem -benchtime=5s .
package bwtsearch

import (
	"bytes"
	"index/suffixarray"
	"os"
	"path/filepath"
	"testing"
)

// ─── corpus loaders ──────────────────────────────────────────────────────────

// benchLoadFile reads a file from the data/ directory and returns its bytes.
// The benchmark is skipped when the file does not exist.
func benchLoadFile(b *testing.B, name string) []byte {
	b.Helper()
	dir, err := os.Getwd()
	if err != nil {
		b.Skip(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "data", name))
	if err != nil {
		b.Skipf("corpus not available (%s): %v — run scripts/download_*.sh", name, err)
	}
	return data
}

// benchLoadGit concatenates all .c and .h files in data/git-src/ into a
// single byte slice (files joined with newline) for use in benchmarks.
func benchLoadGit(b *testing.B) []byte {
	b.Helper()
	dir, err := os.Getwd()
	if err != nil {
		b.Skip(err)
	}
	srcDir := filepath.Join(dir, "data", "git-src")
	if _, err := os.Stat(srcDir); err != nil {
		b.Skip("Git source not found at data/git-src — run ./scripts/download_git.sh")
	}
	texts, _, err := collectSourceFiles(srcDir, map[string]bool{".c": true, ".h": true})
	if err != nil || len(texts) == 0 {
		b.Skipf("collectSourceFiles: %v", err)
	}
	return bytes.Join(texts, []byte("\n"))
}

// ─── build benchmarks ────────────────────────────────────────────────────────
//
// Each sub-benchmark builds a fresh FM-index (or stdlib index) from scratch,
// so b.N reflects the full construction cost.

func runBuildBenchmarks(b *testing.B, text []byte) {
	b.Helper()

	b.Run("Doubling", func(b *testing.B) {
		b.SetBytes(int64(len(text)))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			BuildWithAlgorithm(text, AlgorithmDoubling)
		}
	})

	b.Run("SAIS", func(b *testing.B) {
		b.SetBytes(int64(len(text)))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			BuildWithAlgorithm(text, AlgorithmSAIS)
		}
	})

	b.Run("Stdlib", func(b *testing.B) {
		b.SetBytes(int64(len(text)))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			suffixarray.New(text)
		}
	})
}

func BenchmarkBuild_MobyDick(b *testing.B) {
	runBuildBenchmarks(b, benchLoadFile(b, "moby_dick.txt"))
}

func BenchmarkBuild_Kenshin(b *testing.B) {
	runBuildBenchmarks(b, benchLoadFile(b, "kenshin.txt"))
}

func BenchmarkBuild_Git(b *testing.B) {
	runBuildBenchmarks(b, benchLoadGit(b))
}

// ─── exact-match search benchmarks ───────────────────────────────────────────
//
// The FM-index is built once in the test setup (not timed); only the search
// operation is measured in the b.N loop.
//
// FM-index Count uses O(|P|) backward search.
// stdlib suffixarray Lookup uses O(|P| log n) binary search.

func runExactSearchBenchmarks(b *testing.B, text []byte, patterns []string) {
	b.Helper()
	fmIdx := BuildWithAlgorithm(text, AlgorithmSAIS)
	stdIdx := suffixarray.New(text)

	for _, pat := range patterns {
		pat := pat // avoid loop-variable capture
		p := []byte(pat)

		b.Run("FM/"+pat, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = fmIdx.Count(p)
			}
		})

		b.Run("Stdlib/"+pat, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = stdIdx.Lookup(p, -1)
			}
		})
	}
}

// Moby Dick – exact search patterns (English)
var mobyDickExactPatterns = []string{
	"the",
	"whale",
	"white whale",
	"captain",
	"Ahab",
}

// Kenshin – exact search patterns (Japanese UTF-8)
var kenshinExactPatterns = []string{
	"謙信",
	"上杉謙信",
	"武田信玄",
	"戦国",
	"越後",
}

// Git source – exact search patterns (C/header)
var gitExactPatterns = []string{
	"struct",
	"commit",
	"#include",
	"malloc",
	"diff",
}

func BenchmarkSearchExact_MobyDick(b *testing.B) {
	runExactSearchBenchmarks(b, benchLoadFile(b, "moby_dick.txt"), mobyDickExactPatterns)
}

func BenchmarkSearchExact_Kenshin(b *testing.B) {
	runExactSearchBenchmarks(b, benchLoadFile(b, "kenshin.txt"), kenshinExactPatterns)
}

func BenchmarkSearchExact_Git(b *testing.B) {
	runExactSearchBenchmarks(b, benchLoadGit(b), gitExactPatterns)
}

// ─── star-free regex search benchmarks ───────────────────────────────────────
//
// Only the FM-index supports star-free regular expressions; stdlib
// suffixarray has no regex capability, so it is excluded here.

func runRegexSearchBenchmarks(b *testing.B, text []byte, patterns []string) {
	b.Helper()
	fmIdx := BuildWithAlgorithm(text, AlgorithmSAIS)

	for _, pat := range patterns {
		pat := pat
		b.Run(pat, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = Search(fmIdx, pat, 0)
			}
		})
	}
}

// Moby Dick – star-free regex patterns
var mobyDickRegexPatterns = []string{
	"whale|ship",
	"[Ww]hale",
	"captain|Ahab|Ishmael",
}

// Kenshin – star-free regex patterns (alternation; no ASCII char classes)
var kenshinRegexPatterns = []string{
	"上杉謙信|武田信玄",
	"謙信|信玄",
	"越後|越前",
}

// Git source – star-free regex patterns
var gitRegexPatterns = []string{
	"commit|diff",
	"#include|#define",
	"malloc|free",
}

func BenchmarkSearchRegex_MobyDick(b *testing.B) {
	runRegexSearchBenchmarks(b, benchLoadFile(b, "moby_dick.txt"), mobyDickRegexPatterns)
}

func BenchmarkSearchRegex_Kenshin(b *testing.B) {
	runRegexSearchBenchmarks(b, benchLoadFile(b, "kenshin.txt"), kenshinRegexPatterns)
}

func BenchmarkSearchRegex_Git(b *testing.B) {
	runRegexSearchBenchmarks(b, benchLoadGit(b), gitRegexPatterns)
}
