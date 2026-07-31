# Library API Guide

This repository now provides a public Go package for external users:

`github.com/bgnori/bwt-wheelerlang-study`

Use this package instead of importing `internal/...` packages.
The package also includes executable `go doc` examples in `example_test.go`.

## What is exposed

- `Build`, `BuildWithAlgorithm`, `BuildWithOptions`, `BuildFromFiles`
- `Load`, `ReadFrom`
- `(*Index).Save`, `(*Index).WriteTo`
- `(*Index).Append`, `(*Index).Count`, `(*Index).Locate`, `(*Index).ContextAround`
- `Search` and `Check` for star-free regex search
- `ViolationError` — returned when a pattern violates the star-free constraint
- `UnsupportedError` — returned when a syntactically valid regex construct is unsupported (e.g. position anchors)
- `SuffixArrayAlgorithm`, `AlgorithmDoubling`, `AlgorithmSAIS` — suffix-array algorithm selector
- `OccStructure`, `OccBitvectors`, `OccWaveletTree` — occurrence-array implementation selector
- `Interval` — half-open SA range `[Lo, Hi)` returned by `Search`
- `SearchResult` — result type from `Search`, includes `Intervals`, `TotalCount`, `Truncated`, and `Positions()`
- `WheelerGraphMermaid` for graph visualization

## Install

```bash
go get github.com/bgnori/bwt-wheelerlang-study@latest
```

## Basic usage

```go
package main

import (
	"fmt"
	"log"

	bwtsearch "github.com/bgnori/bwt-wheelerlang-study"
)

func main() {
	text := []byte("abracadabra")
	idx := bwtsearch.Build(text)

	if err := idx.Save("abracadabra.idx"); err != nil {
		log.Fatal(err)
	}

	loaded, err := bwtsearch.Load("abracadabra.idx")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("count(abra):", loaded.Count([]byte("abra")))
}
```

## Build from multiple files

`BuildFromFiles` concatenates several byte slices (e.g. multiple documents)
with a separator and builds a single FM-index over the combined corpus.

```go
texts := [][]byte{
	[]byte("hello world"),
	[]byte("world peace"),
}
idx := bwtsearch.BuildFromFiles(texts, []byte("\n"))
fmt.Println(idx.Count([]byte("world"))) // 2
```

A `nil` separator defaults to `"\n"`.  The separator must not contain `0x00`
(reserved as the FM-index sentinel).

## Incremental update (RopeBWT style)

`Append` extends an existing index with additional bytes while preserving the
original build options (suffix-array algorithm and occurrence structure).

```go
idx := bwtsearch.BuildWithOptions([]byte("hello"), bwtsearch.AlgorithmSAIS, bwtsearch.OccWaveletTree)
_ = idx.Append([]byte(" world"))
fmt.Println(idx.Count([]byte("hello world"))) // 1
```

## Star-free regex search

```go
res, err := bwtsearch.Search(idx, "ab?ra|cad", 20)
if err != nil {
	// Distinguish star-free violations from regex syntax errors.
	var ve *bwtsearch.ViolationError
	if errors.As(err, &ve) {
		log.Fatalf("star-free violation: %v", ve)
	}
	log.Fatal(err)
}

positions := res.Positions(idx)
fmt.Println("total:", res.TotalCount, "positions:", positions)
```

Patterns with `*`, `+`, or `{n,}` are rejected and return a `*ViolationError`.
Position anchors (`^`, `$`, `\b`, `\B`) are not supported by FM-index backward
search and return a `*UnsupportedError` instead of being silently ignored.

**Known limitation — character classes are ASCII-only:** `evalCharClass` and
`evalAnyChar` only process runes in the ASCII range (U+0000–U+007F).  A pattern
such as `[あ-を]` returns 0 results without an error.  Use literal UTF-8 byte
patterns or alternation (e.g. `上杉謙信|武田信玄`) for non-ASCII matching.

## Choose suffix-array algorithm

```go
idx := bwtsearch.BuildWithAlgorithm(text, bwtsearch.AlgorithmSAIS)
```

Available algorithms:

- `AlgorithmDoubling` (default) — O(n log² n), moderate memory
- `AlgorithmSAIS` — O(n) linear time, recommended for large texts

## Choose occurrence-array structure

`BuildWithOptions` lets you select both the suffix-array algorithm and the
occurrence-array implementation:

```go
idx := bwtsearch.BuildWithOptions(text, bwtsearch.AlgorithmSAIS, bwtsearch.OccWaveletTree)
```

Available occurrence structures:

- `OccBitvectors` (default) — one succinct bit-vector per distinct character;
  O(1) rank queries; on-disk format `FMIDX01`.
- `OccWaveletTree` — a Wavelet Tree over the BWT; O(log σ) rank queries and
  O(n log σ) total space.  Advantageous when the alphabet is large or nearly all
  256 byte values appear; on-disk format `FMIDX02`.

## Generate Wheeler graph (Mermaid)

```go
graph := idx.WheelerGraphMermaid(20)
fmt.Println(graph)
```

## Versioning and stability

- Treat the top-level package as the **stable public API**.
- `internal/...` packages are implementation details and may change without
  notice; do not import them directly.
- The package follows [Semantic Versioning](https://semver.org/). Use the
  `@latest` tag or pin a specific `@vX.Y.Z` tag for reproducible builds.
- `SearchResult.Truncated` and `SearchResult.TotalCount` are part of the
  stable API; inspect them when `limit > 0` to detect truncation.

## Error handling summary

| Condition | Error type |
|---|---|
| Pattern uses `*`, `+`, or `{n,}` | `*bwtsearch.ViolationError` |
| Pattern uses anchors (`^`, `$`, `\b`, `\B`) | `*bwtsearch.UnsupportedError` |
| Pattern has invalid regex syntax | wrapped `*syntax.Error` from `regexp/syntax` |
| `nil` Index passed to Search | plain `error` string |
