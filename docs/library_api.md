# Library API Guide

公開パッケージ:

`github.com/bgnori/textindex`

`internal/...` の直接 import ではなく、上記パッケージを利用してください。

## 主な公開 API

### FM-index

- `Build`, `BuildWithAlgorithm`, `BuildWithOptions`
- `BuildFromFiles`, `BuildFromFilesWithOptions`
- `Load`, `ReadFrom`
- `(*Index).Save`, `(*Index).WriteTo`
- `(*Index).Append`, `(*Index).Count`, `(*Index).Locate`, `(*Index).ContextAround`
- `(*Index).SALen`, `(*Index).AlphabetSize`, `(*Index).OccType`, `(*Index).NumBWTRuns`
- `Search`, `Check`

### Suffix Array（標準ライブラリ）

- `BuildStdlib`, `BuildStdlibFromFiles`
- `LoadStdlib`, `ReadStdlibFrom`
- `(*StdlibIndex).Save`, `(*StdlibIndex).WriteTo`
- `(*StdlibIndex).Count`, `(*StdlibIndex).Locate`, `(*StdlibIndex).ContextAround`

### 双方向 FM-index

- `BuildBi`, `BuildBiWithOptions`
- `BuildBiFromFiles`, `BuildBiFromFilesWithOptions`
- `LoadBi`, `ReadBiFrom`
- `(*BiIndex).Save`, `(*BiIndex).WriteTo`
- `(*BiIndex).Count`, `(*BiIndex).Locate`, `(*BiIndex).ContextAround`
- `(*BiIndex).FullInterval`, `(*BiIndex).ExtendLeft`, `(*BiIndex).ExtendRight`

### 型・定数

- `ViolationError`（星なし制約違反）
- `UnsupportedError`（未対応正規表現構文）
- `SuffixArrayAlgorithm`: `AlgorithmDoubling`, `AlgorithmSAIS`
- `OccStructure`: `OccBitvectors`, `OccWaveletTree`, `OccWaveletMatrix`, `OccRLBWT`
- `Interval`, `SearchResult`

## 最小例

```go
package main

import (
	"fmt"
	"log"

	bwtsearch "github.com/bgnori/textindex"
)

func main() {
	idx := bwtsearch.Build([]byte("abracadabra"))
	if err := idx.Save("abracadabra.idx"); err != nil {
		log.Fatal(err)
	}

	loaded, err := bwtsearch.Load("abracadabra.idx")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(loaded.Count([]byte("abra")))
}
```

## 検索仕様（重要）

- `Search` は **FM-index (`*Index`) 専用** です。
- `Search` のパターンは星なし正規表現として扱われ、`*`, `+`, `{n,}` は `ViolationError` になります。
- アンカー（`^`, `$`, `\b`, `\B`）など後方検索で未対応の構文は `UnsupportedError` になります。
- 文字クラス（`[a-z]`、`.`、`\w` など）は **ASCII (U+0000–U+007F) のみ対応** です。128 以上のルーンを含む文字クラス（例: `[ぁ-ん]`、`[α-ω]`）はエラーにならず 0 件を返します。非 ASCII 文字はリテラル（UTF-8 バイト列そのまま）としてのみ検索できます。
- `StdlibIndex` / `BiIndex` は `Count` / `Locate` でリテラル検索します。

## 複数ファイル入力

`BuildFromFiles*` / `BuildStdlibFromFiles` / `BuildBiFromFiles*` は、`separator=nil` の場合に改行 (`\n`) 連結を使います。separator に `0x00` を含めることはできません。

## 永続化形式

`Save` / `WriteTo` は Occ 構造に応じて `FMIDX05`（bitvectors）、`FMIDX06`（Wavelet Tree）、
`FMIDX07`（Wavelet Matrix）、`FMIDX08`（RLBWT）を使用し、構築時の suffix-array
アルゴリズムも保存します。`FMIDX01`〜`FMIDX04` は従来どおり doubling として読み込まれます。
