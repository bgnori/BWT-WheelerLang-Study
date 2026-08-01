# Library API Guide

公開パッケージ:

`github.com/bgnori/textindex`

`internal/...` の直接 import ではなく、上記パッケージを利用してください。

## 主な公開 API

### FM-index

- `Build`, `BuildWithAlgorithm`, `BuildWithOptions`
- `BuildWithConfig`
- `BuildFromFiles`, `BuildFromFilesWithOptions`, `BuildFromFilesWithConfig`
- `Load`, `ReadFrom`
- `(*Index).Save`, `(*Index).WriteTo`
- `(*Index).Append`, `(*Index).Count`, `(*Index).Locate`, `(*Index).ContextAround`
- `(*Index).SALen`, `(*Index).AlphabetSize`, `(*Index).OccType`, `(*Index).OccStorage`, `(*Index).NumBWTRuns`
- `Search`, `Check`

`Build` / `BuildFromFiles` の既定値は `AlgorithmSAIS + OccRLBWT` です。

### Suffix Array（標準ライブラリ）

- `BuildStdlib`, `BuildStdlibFromFiles`
- `LoadStdlib`, `ReadStdlibFrom`
- `(*StdlibIndex).Save`, `(*StdlibIndex).WriteTo`
- `(*StdlibIndex).Count`, `(*StdlibIndex).Locate`, `(*StdlibIndex).ContextAround`

### 双方向 FM-index

- `BuildBi`, `BuildBiWithOptions`
- `BuildBiWithConfig`
- `BuildBiFromFiles`, `BuildBiFromFilesWithOptions`, `BuildBiFromFilesWithConfig`
- `LoadBi`, `ReadBiFrom`
- `(*BiIndex).Save`, `(*BiIndex).WriteTo`
- `(*BiIndex).Count`, `(*BiIndex).Locate`, `(*BiIndex).ContextAround`
- `(*BiIndex).FullInterval`, `(*BiIndex).ExtendLeft`, `(*BiIndex).ExtendRight`

`BuildBi` / `BuildBiFromFiles` の既定値は `AlgorithmSAIS + OccRLBWT` です。

### 型・定数

- `ViolationError`（星なし制約違反）
- `UnsupportedError`（未対応正規表現構文）
- `SuffixArrayAlgorithm`: `AlgorithmDoubling`, `AlgorithmSAIS`
- `OccStructure`: `OccBitvectors`, `OccWaveletTree`, `OccWaveletMatrix`, `OccRLBWT`, `OccRRR`, `OccEliasFano`, `OccPoppy`, `OccDynamicBitvectors`, `OccExternalWaveletTree`（互換用）
- `OccStorageMode`: `OccStorageInMemory`, `OccStorageExternal`
- `OccExternalStrategy`: `OccExternalStrategyLSM`, `OccExternalStrategyBPlusTree`, `OccExternalStrategyInvertedSegments`
- `OccStorageOptions`: `Mode`, `DiskBlockSize`, `ExternalStrategy`
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

`Save` / `WriteTo` は 7 バイトの Magic として `FMI` + 2 桁の Occ 構造 ID + 2 桁の永続化戦略 ID を使います。
Occ 構造 ID は 01=bitvectors, 02=Wavelet Tree, 03=Wavelet Matrix, 04=RLBWT, 05=RRR, 06=Elias-Fano, 07=Poppy, 08=Dynamic Bit Vector です。
永続化戦略 ID は 01=inline occ（bitvectors 専用）, 02=rebuild occ（BWT から再構築）, 03=external LSM, 04=external B+Tree, 05=external Inverted Segments です。
たとえば `FMI0203` は Wavelet Tree + external LSM、`FMI0104` は bitvectors + external B+Tree を表します。どの external 形式でも disk block size はヘッダに保存されます。
後方互換読み込みは提供しません。
