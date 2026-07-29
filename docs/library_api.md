# Library API Guide

This repository now provides a public Go package for external users:

`github.com/bgnori/bwt-wheelerlang-study`

Use this package instead of importing `internal/...` packages.
The package also includes executable `go doc` examples in `example_test.go`.

## What is exposed

- `Build`, `BuildWithAlgorithm`
- `Load`, `ReadFrom`
- `(*Index).Save`, `(*Index).WriteTo`
- `(*Index).Count`, `(*Index).Locate`, `(*Index).ContextAround`
- `Search` and `Check` for star-free regex search
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

## Star-free regex search

```go
res, err := bwtsearch.Search(idx, "ab?ra|cad", 20)
if err != nil {
	// Includes star-free validation errors and regex syntax errors.
	log.Fatal(err)
}

positions := res.Positions(idx)
fmt.Println("total:", res.TotalCount, "positions:", positions)
```

Patterns with `*`, `+`, or `{n,}` are rejected by design.

## Choose suffix-array algorithm

```go
idx := bwtsearch.BuildWithAlgorithm(text, bwtsearch.AlgorithmSAIS)
```

Available algorithms:

- `AlgorithmDoubling` (default)
- `AlgorithmSAIS`

## Generate Wheeler graph (Mermaid)

```go
graph := idx.WheelerGraphMermaid(20)
fmt.Println(graph)
```

## Stability notes

- Treat the top-level package as the stable public API.
- `internal/...` packages are implementation details and may change.