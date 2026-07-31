// Package bwtsearch – performance benchmarks.
//
// These benchmarks cover:
//   - suffix-array construction algorithms (Doubling / SA-IS / stdlib)
//   - FM-index Occ structures (bitvectors / wavelet tree / wavelet matrix / RLBWT)
//   - bidirectional FM-index (BiFM-index)
//   - exact and star-free regex search where supported
//
// # Run
//
//	go test -bench=. -benchmem -benchtime=3s .
package bwtsearch

import (
	"bytes"
	"index/suffixarray"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── dataset loaders ─────────────────────────────────────────────────────────

type benchDataset struct {
	name          string
	text          []byte
	exactPatterns []string
	regexPatterns []string
}

func benchReadDataFile(name string) ([]byte, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, false
	}
	b, err := os.ReadFile(filepath.Join(dir, "data", name))
	if err != nil || len(b) == 0 {
		return nil, false
	}
	return b, true
}

func benchReadGit() ([]byte, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, false
	}
	srcDir := filepath.Join(dir, "data", "git-src")
	texts, _, err := collectSourceFiles(srcDir, map[string]bool{".c": true, ".h": true})
	if err != nil || len(texts) == 0 {
		return nil, false
	}
	return bytes.Join(texts, []byte("\n")), true
}

func benchSyntheticCorpus(block string, size int) []byte {
	if size <= 0 {
		return nil
	}
	var sb strings.Builder
	sb.Grow(size + len(block))
	for sb.Len() < size {
		sb.WriteString(block)
		sb.WriteByte('\n')
	}
	out := sb.String()
	return []byte(out[:size])
}

func loadSmallDataset() benchDataset {
	if data, ok := benchReadDataFile("kenshin.txt"); ok && len(data) <= 1_000_000 {
		return benchDataset{
			name: "Kenshin<=1MB",
			text: data,
			exactPatterns: []string{
				"謙信", "上杉謙信", "武田信玄", "戦国", "越後",
			},
			regexPatterns: []string{
				"上杉謙信|武田信玄", "謙信|信玄", "越後|越前",
			},
		}
	}
	return benchDataset{
		name: "SyntheticJA-768KB",
		text: benchSyntheticCorpus(
			"上杉謙信 武田信玄 戦国 越後 越前 謙信 謙信 上杉謙信",
			768*1024,
		),
		exactPatterns: []string{
			"謙信", "上杉謙信", "武田信玄", "戦国", "越後",
		},
		regexPatterns: []string{
			"上杉謙信|武田信玄", "謙信|信玄", "越後|越前",
		},
	}
}

func loadReasonableDataset() benchDataset {
	if data, ok := benchReadGit(); ok && len(data) > 1_000_000 {
		return benchDataset{
			name: "GitSource",
			text: data,
			exactPatterns: []string{
				"struct", "commit", "#include", "malloc", "diff",
			},
			regexPatterns: []string{
				"commit|diff", "#include|#define", "malloc|free",
			},
		}
	}
	return benchDataset{
		name: "SyntheticCode-8MB",
		text: benchSyntheticCorpus(
			"static int commit_diff(struct repo *r) { #include <stdlib.h> if (malloc(64)) return 0; }",
			8*1024*1024,
		),
		exactPatterns: []string{
			"struct", "commit", "#include", "malloc", "diff",
		},
		regexPatterns: []string{
			"commit|diff", "#include|#define", "malloc|free",
		},
	}
}

func loadGeneratedLogDataset() benchDataset {
	if data, ok := benchReadDataFile("fake-logs/flog_apache_common.log"); ok && len(data) > 0 {
		return benchDataset{
			name: "GeneratedLog-ApacheCommon",
			text: data,
			exactPatterns: []string{
				"GET", "HTTP/1.1", " 200 ", "Mozilla", "/",
			},
			regexPatterns: []string{
				"GET|POST", "HTTP/1.0|HTTP/1.1", " 200 | 404 ",
			},
		}
	}
	return benchDataset{
		name: "SyntheticLog-5MB",
		text: benchSyntheticCorpus(
			`127.0.0.1 - - [01/Jan/2026:00:00:00 +0000] "GET /index.html HTTP/1.1" 200 1234 "-" "Mozilla/5.0"`,
			5*1024*1024,
		),
		exactPatterns: []string{
			"GET", "HTTP/1.1", " 200 ", "Mozilla", "/",
		},
		regexPatterns: []string{
			"GET|POST", "HTTP/1.0|HTTP/1.1", " 200 | 404 ",
		},
	}
}

func loadGenomeDataset() benchDataset {
	if data, ok := benchReadDataFile("ecoli.txt"); ok && len(data) > 0 {
		return benchDataset{
			name: "Genome-Ecoli",
			text: data,
			exactPatterns: []string{
				"ATGAAACGC", "GTTACCTGCC", "CGCGCG", "TTTTTT", "AGCTTTTC",
			},
			regexPatterns: []string{
				"ATGAAACGC|GTTACCTGCC", "CGCGCG|GCGCGC", "TTTTTT|AAAAAA",
			},
		}
	}
	return benchDataset{
		name: "SyntheticGenome-4MB",
		text: benchSyntheticCorpus(
			"AGCTTTTCATTCTGACTGCAACGGGCAATATGTCTCTGTGTGGATTAAAAAAAGAGTGTCTGATAGCAGCTTCTGAAC",
			4*1024*1024,
		),
		exactPatterns: []string{
			"ATGAAACGC", "GTTACCTGCC", "CGCGCG", "TTTTTT", "AGCTTTTC",
		},
		regexPatterns: []string{
			"ATGAAACGC|GTTACCTGCC", "CGCGCG|GCGCGC", "TTTTTT|AAAAAA",
		},
	}
}

// ─── benchmark sections ──────────────────────────────────────────────────────

func runBuildBenchmarks(b *testing.B, ds benchDataset) {
	b.Helper()
	text := ds.text

	b.Run(ds.name, func(b *testing.B) {
		b.Run("FM/Doubling+Bitvectors", func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				BuildWithOptions(text, AlgorithmDoubling, OccBitvectors)
			}
		})

		b.Run("FM/SAIS+Bitvectors", func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				BuildWithOptions(text, AlgorithmSAIS, OccBitvectors)
			}
		})

		b.Run("FM/SAIS+WaveletTree", func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				BuildWithOptions(text, AlgorithmSAIS, OccWaveletTree)
			}
		})

		b.Run("FM/SAIS+WaveletMatrix", func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				BuildWithOptions(text, AlgorithmSAIS, OccWaveletMatrix)
			}
		})

		b.Run("FM/SAIS+RLBWT", func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				BuildWithOptions(text, AlgorithmSAIS, OccRLBWT)
			}
		})

		b.Run("BiFM/SAIS+Bitvectors", func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				BuildBiWithOptions(text, AlgorithmSAIS, OccBitvectors)
			}
		})

		b.Run("Stdlib/SuffixArray", func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				suffixarray.New(text)
			}
		})
	})
}

func runExactSearchBenchmarks(b *testing.B, ds benchDataset) {
	b.Helper()
	fmBitVec := BuildWithOptions(ds.text, AlgorithmSAIS, OccBitvectors)
	fmWavelet := BuildWithOptions(ds.text, AlgorithmSAIS, OccWaveletTree)
	fmWm := BuildWithOptions(ds.text, AlgorithmSAIS, OccWaveletMatrix)
	fmRlbwt := BuildWithOptions(ds.text, AlgorithmSAIS, OccRLBWT)
	bi := BuildBiWithOptions(ds.text, AlgorithmSAIS, OccBitvectors)
	stdIdx := suffixarray.New(ds.text)

	b.Run(ds.name, func(b *testing.B) {
		for _, pat := range ds.exactPatterns {
			pat := pat
			p := []byte(pat)

			b.Run("FM/Bitvectors/"+pat, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = fmBitVec.Count(p)
				}
			})

			b.Run("FM/WaveletTree/"+pat, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = fmWavelet.Count(p)
				}
			})

			b.Run("FM/WaveletMatrix/"+pat, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = fmWm.Count(p)
				}
			})

			b.Run("FM/RLBWT/"+pat, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = fmRlbwt.Count(p)
				}
			})

			b.Run("BiFM/Bitvectors/"+pat, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = bi.Count(p)
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
	})
}

func runRegexSearchBenchmarks(b *testing.B, ds benchDataset) {
	b.Helper()
	fmBitVec := BuildWithOptions(ds.text, AlgorithmSAIS, OccBitvectors)
	fmWavelet := BuildWithOptions(ds.text, AlgorithmSAIS, OccWaveletTree)
	fmWm := BuildWithOptions(ds.text, AlgorithmSAIS, OccWaveletMatrix)
	fmRlbwt := BuildWithOptions(ds.text, AlgorithmSAIS, OccRLBWT)

	b.Run(ds.name, func(b *testing.B) {
		for _, pat := range ds.regexPatterns {
			pat := pat

			b.Run("FM/Bitvectors/"+pat, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = Search(fmBitVec, pat, 0)
				}
			})

			b.Run("FM/WaveletTree/"+pat, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = Search(fmWavelet, pat, 0)
				}
			})

			b.Run("FM/WaveletMatrix/"+pat, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = Search(fmWm, pat, 0)
				}
			})

			b.Run("FM/RLBWT/"+pat, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = Search(fmRlbwt, pat, 0)
				}
			})
		}
	})
}

// ─── exported benchmark entrypoints ──────────────────────────────────────────

func BenchmarkBuild_SmallUnder1MB(b *testing.B) {
	runBuildBenchmarks(b, loadSmallDataset())
}

func BenchmarkBuild_ReasonableSize(b *testing.B) {
	runBuildBenchmarks(b, loadReasonableDataset())
}

func BenchmarkSearchExact_SmallUnder1MB(b *testing.B) {
	runExactSearchBenchmarks(b, loadSmallDataset())
}

func BenchmarkSearchExact_ReasonableSize(b *testing.B) {
	runExactSearchBenchmarks(b, loadReasonableDataset())
}

func BenchmarkSearchRegex_SmallUnder1MB(b *testing.B) {
	runRegexSearchBenchmarks(b, loadSmallDataset())
}

func BenchmarkSearchRegex_ReasonableSize(b *testing.B) {
	runRegexSearchBenchmarks(b, loadReasonableDataset())
}

func BenchmarkBuild_GeneratedLogs(b *testing.B) {
	runBuildBenchmarks(b, loadGeneratedLogDataset())
}

func BenchmarkBuild_Genome(b *testing.B) {
	runBuildBenchmarks(b, loadGenomeDataset())
}

func BenchmarkSearchExact_GeneratedLogs(b *testing.B) {
	runExactSearchBenchmarks(b, loadGeneratedLogDataset())
}

func BenchmarkSearchExact_Genome(b *testing.B) {
	runExactSearchBenchmarks(b, loadGenomeDataset())
}

func BenchmarkSearchRegex_GeneratedLogs(b *testing.B) {
	runRegexSearchBenchmarks(b, loadGeneratedLogDataset())
}

func BenchmarkSearchRegex_Genome(b *testing.B) {
	runRegexSearchBenchmarks(b, loadGenomeDataset())
}
