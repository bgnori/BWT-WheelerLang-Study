# プロジェクトレビュー — Go 外部ライブラリへの移行評価

レビュー日: 2026-07-31

## 概要

このドキュメントは、`github.com/bgnori/textindex` を
勉強用プロジェクトから Go 外部ライブラリとして公開するために実施したコードレビューの記録です。

---

## Go 外部ライブラリ要件チェックリスト

| 要件 | 状態 | 備考 |
|------|------|------|
| モジュールパスが GitHub リポジトリパスと一致する | ✅ | `github.com/bgnori/textindex` |
| 公開 API がトップレベルパッケージに集約されている | ✅ | `api.go` (`package bwtsearch`) |
| 内部実装が `internal/` に隔離されている | ✅ | `internal/fmindex`, `internal/bitvector`, `internal/starfree` |
| パッケージコメント（godoc）が記述されている | ✅ | 全パッケージに記述済み |
| 公開シンボルにコメントが付いている | ✅ | 全エクスポートに docstring あり |
| runnable な Example テストが存在する | ✅ | `example_test.go` に 8 例 |
| ライセンスファイルが存在する | ✅ | `LICENSE` (MIT) |
| `go.mod` と `go.sum` が最新状態 | ✅ | go 1.21 |
| `go vet` / `go test -race` がクリア | ✅ | 全パッケージ PASS |
| `io.WriterTo` 契約を満たしている | ✅ | **本レビューで修正** |
| エラー型が公開 API から型アサート可能 | ✅ | **本レビューで追加** |
| `BuildFromFiles` がドキュメントに記載されている | ✅ | **本レビューで追加** |
| `BuildWithOptions` / `OccWaveletTree` がドキュメントに記載されている | ✅ | **ウェーブレット木追加時に反映** |
| `Append` (インクリメンタル更新) がドキュメントに記載されている | ✅ | **RopeBWT 追加時に反映** |

---

## 発見された問題と対処

### 1. `WriteTo` が常に 0 バイトを返していた（バグ）

**問題:**  
`internal/fmindex.Index.WriteTo` は `io.WriterTo` インターフェースを実装しているが、
書き込んだバイト数を常に `0` として返していた（コメント: `// byte count omitted`）。
`io.WriterTo` の契約では、実際に書き込んだバイト数を返すことが求められる。

**影響範囲:**  
- `(*Index).WriteTo` (公開 API)  
- `(*Index).Save` (内部で `WriteTo` を使用)  
- `bufio.Writer` でのバッファリングにより、呼び出し元が進捗を監視できない

**修正:**  
`internal/fmindex/fmindex.go` に `countingWriter` 型を追加し、
`bufio.Writer` のラッパーとして使用することで実際のバイト数を追跡するようにした。

```go
type countingWriter struct {
    w io.Writer
    n int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
    n, err := cw.w.Write(p)
    cw.n += int64(n)
    return n, err
}
```

---

### 2. `ViolationError` が公開 API で型アサートできなかった

**問題:**  
`Check` / `Search` が返すエラーは、内部の `starfree.ViolationError` を
`error` インターフェースとして返していたが、呼び出し元は
`*starfree.ViolationError` を import できない（`internal` パッケージのため）。
そのため、星なし制約違反と正規表現構文エラーをプログラム的に区別できなかった。

**修正:**  
公開パッケージ (`api.go`) に `bwtsearch.ViolationError` 型を追加した。
`Check` / `Search` は内部の `starfree.ViolationError` を受け取った際に
公開型に変換して返すようにした。

```go
var ve *bwtsearch.ViolationError
if errors.As(err, &ve) {
    // 星なし制約違反として処理
}
```

---

### 3. `BuildFromFiles` がライブラリドキュメントに未記載

**問題:**  
`api.go` には `BuildFromFiles` が実装・エクスポートされているが、
`docs/library_api.md` には記載がなかった。

**修正:**  
`docs/library_api.md` に `BuildFromFiles` のセクションを追加し、
使用例とセパレータの制約（`0x00` 禁止）を記載した。

---

## 既知の制限事項（未修正）

以下は本レビューでは変更しないが、今後の検討事項として記録する。

### 文字クラス検索の ASCII 限定

`evalCharClass` と `evalAnyChar` は ASCII 範囲 (0–127) の文字にのみ対応する。
Unicode 文字クラス (`\p{Han}` 等) はサポートしない。
FM-index がバイト列上で動作するため、マルチバイト文字の範囲検索は
UTF-8 エンコードを意識した実装が必要になり、追加の設計が必要。

### `WriteTo` の返却バイト数の正確性

`WriteTo` が返すバイト数は、`bufio.Writer` がアンダーライングライターに
フラッシュした実バイト数である。書き込みエラーが途中で発生した場合は
実際に書き込まれた量より小さい値が返ることがある（仕様通り）。

### モジュール名に "study" が含まれる

モジュールパス `github.com/bgnori/bwt-wheelerlang-study` には元々の
「勉強用」という文脈の名称が含まれている。外部ライブラリとして広く公開する
場合はモジュールパスの変更（`v2` への移行と同様の手順）を検討してもよいが、
後方互換性を壊すため慎重に判断すること。

### TUI 依存関係 (`charmbracelet`) のライブラリ影響

`go.mod` に `charmbracelet/bubbletea`, `charmbracelet/bubbles` が direct 依存として
記載されているが、これらは `cmd/textindex/main.go` のみで使用する。
Go モジュールシステムはパッケージ単位で依存関係を解決するため、
ライブラリパッケージのみを import するユーザーにはこれらの依存は伝播しない。
現状では実害がないが、将来的に `go.mod` の依存を最小化したい場合は
`cmd/textindex` を別モジュールに切り出す方法がある。

---

## セマンティックバージョニングの推奨

外部ライブラリとして安定運用するために、以下の運用を推奨する:

1. **初回リリース**: `git tag v0.1.0` または `v1.0.0` を打つ
2. **CHANGELOG.md**: リリースごとに変更点を記録する
3. **後方互換性**: `v1.x.x` 期間中は公開 API の破壊的変更を避ける
4. **`go get` での利用**: `go get github.com/bgnori/textindex@v1.0.0`

---

## まとめ

本レビューで修正・追加した内容:

| 変更ファイル | 内容 |
|---|---|
| `internal/fmindex/fmindex.go` | `countingWriter` 追加、`WriteTo` のバイト数を正確に返すよう修正 |
| `api.go` | `ViolationError` 型を公開 API に追加、`Check`/`Search` でエラーを変換 |
| `docs/library_api.md` | `BuildFromFiles`、エラー型、バージョニングの説明を追加 |
| `docs/review.md` | 本ドキュメント（新規作成） |

その後の機能追加で更新した内容:

| 変更ファイル | 内容 |
|---|---|
| `internal/wavelet/` | Wavelet Tree 実装を新規追加 |
| `internal/fmindex/occ.go` | `OccWaveletTree` オプションを追加、FMIDX02 フォーマット対応 |
| `internal/fmindex/rope.go` | RopeBWT スタイルのインクリメンタル `Append` を追加 |
| `api.go` | `BuildWithOptions`、`OccStructure`/`OccBitvectors`/`OccWaveletTree`、`UnsupportedError`、`Interval`、`SearchResult`、`WriteTo`、`Append` を追加 |
| `example_test.go` | `ExampleBuild_japanese`、`ExampleBuildWithOptions`、`ExampleIndex_Append` を追加（計 8 例） |
| `git_source_test.go` | `BuildFromFiles` を使ったマルチファイルインデックスの統合テストを追加 |
| `bench_test.go` | 3 データセット × 3 アルゴリズムのベンチマークスイートを追加 |
| `docs/library_api.md` | `BuildWithOptions`、`OccStructure`、`WriteTo`、型一覧を追記 |
| `README.md` | API 一覧・アーキテクチャ図に `wavelet/` および新規 API を反映 |
