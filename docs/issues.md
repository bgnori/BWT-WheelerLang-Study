# 課題一覧 — ライブラリ公開レビューで洗い出した問題点

作成日: 2026-07-31  
最終更新: 2026-07-31  
対象: `github.com/bgnori/bwt-wheelerlang-study` の外部ライブラリ化に向けたレビュー

---

## 優先度について

| 優先度 | 説明 |
|--------|------|
| 🔴 High | 正確性・安全性に関わる問題。公開前に修正推奨 |
| 🟡 Medium | API 品質・ユーザー体験に関わる問題 |
| 🟢 Low | 将来的な検討事項・設計改善 |

---

## #1 — 位置アンカーがサイレントに無視される 🔴 ✅ 対応済み

**ファイル:** `internal/starfree/starfree.go`

**問題:**  
`evalRegex` では `OpBeginText`、`OpEndText`、`OpBeginLine`、`OpEndLine`、`OpWordBoundary`、`OpNoWordBoundary` を「全区間を返す（≒ マッチしない条件を無視）」として処理しています。

そのため `^hello` を検索すると `hello` と同じ全マッチが返り、ユーザーが意図しない結果になります。

**再現:**
```go
idx := bwtsearch.Build([]byte("hello\nworld\nhello"))
res, _ := bwtsearch.Search(idx, "^hello", 0)
// 期待: 行頭の hello のみ
// 実際: hello の全出現 (2件) が返る
```

**対処案:**
- アンカーを含むパターンに対して `ViolationError`（または新しい `UnsupportedError`）を返す
- 少なくとも `docs/library_api.md` に「位置アンカーは FM-index の後方検索モデルでは表現できないため無視される」と明記する

**対応内容:**  
`api.go` に `UnsupportedError` 型を追加し、`internal/starfree/starfree.go` の `checkNode()` で
`OpBeginText`・`OpEndText`・`OpBeginLine`・`OpEndLine`・`OpWordBoundary`・`OpNoWordBoundary` を
検出した際に `UnsupportedError` を返すよう修正済み。`Check` および `Search` 経由でも正しく伝播する。

---

## #2 — `BuildFromFiles` のセパレータに `0x00` を渡してもエラーにならない 🔴 ✅ 対応済み

**ファイル:** `api.go`

**問題:**  
ドキュメントには「セパレータに `0x00` を含めてはならない（FM-index の番兵として予約済み）」と記載されているが、実行時チェックがない。

`BuildFromFiles(texts, []byte{0})` を呼び出すと、インデックスが壊れ、以後の検索結果が不正確になる可能性があります。

**対処案:**
```go
for _, b := range separator {
    if b == 0 {
        panic("bwtsearch: separator must not contain 0x00 (reserved as sentinel)")
    }
}
```
または `error` を返すシグネチャに変更する（破壊的変更になるため `v2` 以降）。

**対応内容:**  
`api.go` の `BuildFromFiles` に `0x00` バイトの検証ループを追加し、セパレータに `0x00` が含まれる場合は
`"bwtsearch: separator must not contain 0x00 (reserved as sentinel)"` メッセージとともに panic するよう修正済み。
godoc にも panic 条件を明記した。`api_test.go` に 2 件のテストケースを追加して動作を検証済み。

---

## #3 — `*Index` メソッド群の nil チェックが非一貫 🔴 ✅ 対応済み

**ファイル:** `api.go`

**問題:**  
`WriteTo`、`Append`、`Search`（公開 API 側）は `idx == nil || idx.inner == nil` をチェックするが、  
`Count`、`Locate`、`TextLen`、`SALen`、`SAAt`、`AlphabetSize`、`BWT` は nil チェックなしで panic します。

```go
var idx *bwtsearch.Index
idx.Count([]byte("abc"))  // → panic: runtime error: nil pointer dereference
```

**現在の状態（部分対応）:**  
`TestNilIndexErrors` で `Search`、`Append`、`WriteTo` の nil チェックを検証するテストが追加されています。
ただし `Count`、`Locate`、`TextLen` 等の nil チェックは未追加のため、問題は完全には解消されていません。

**対処案:**  
以下のいずれかに統一する：
1. ゼロ値 `*Index` に対してはゼロ値を返す（`Count` → 0、`Locate` → nil、`BWT` → nil など）
2. panic を "documented behavior" として godoc に明記する

**対応内容:**  
対処案 1 を採用。`Count`・`Locate`・`TextLen`・`SALen`・`SAAt`・`AlphabetSize`・`NumBWTRuns`・
`OccType`・`BWT`・`ContextAround`・`WheelerGraphMermaid` に nil チェックを追加し、
nil レシーバーに対してゼロ値（0 / nil / 空文字列）を返すよう統一した。各メソッドの godoc にも明記。
`TestNilIndexZeroValues` で全メソッドの動作を検証済み。

---

## #4 — Unicode 文字クラスがエラーなく 0 件になる 🟡 ✅ 対応済み

**ファイル:** `internal/starfree/starfree.go`

**問題:**  
`evalCharClass` と `evalAnyChar` は ASCII 範囲 (0–127) のルーンのみを処理し、128 以上のルーンを無視します。

そのため `[ぁ-ん]` や `[α-ω]` を検索するとエラーなしに 0 件が返り、利用者が混乱する可能性があります。

```go
idx := bwtsearch.Build([]byte("ひらがなてきすと"))
res, err := bwtsearch.Search(idx, "[あ-を]", 0)
// err == nil, res.TotalCount == 0  ← 正しいが理由が不明
```

**対処案:**  
- Unicode 範囲（rune > 127）を含む文字クラスに対して `UnsupportedError` を返す
- または `docs/library_api.md` に「文字クラスは ASCII (U+0000–U+007F) のみ対応」と明記する

**現在の状態（未対応）:**  
再レビュー（2026-07-31）で確認したところ、`docs/library_api.md` に ASCII 限定の制限事項の記載は
存在しない（以前の「部分対応」記載は誤り）。`UnsupportedError` を返す対応も未実施のため、
ドキュメント追記またはエラー返却のいずれかの対応が必要。

**対応内容:**  
`docs/library_api.md` の「検索仕様（重要）」に、文字クラスは ASCII (U+0000–U+007F) のみ対応であり、
非 ASCII を含む文字クラスはエラーなしに 0 件を返す旨と、非 ASCII 文字はリテラルとしてのみ検索
できる旨を明記した。

---

## #5 — `Search` が正規表現を 2 回パースしていた 🟡 ✅ 対応済み

**ファイル:** `internal/starfree/starfree.go`

**問題:**  
`Search` の内部で `Check(pattern)` → `syntax.Parse #1` を呼び、  
次に `syntax.Parse(pattern)` を直接呼ぶ（#2）という二重パースが発生しています。

頻繁に検索が発生するユースケースでは余分な解析コストがかかります。

```go
func Search(idx *fmindex.Index, pattern string, limit int) (*SearchResult, error) {
    if err := Check(pattern); err != nil {  // ← syntax.Parse #1
        return nil, err
    }
    re, err := syntax.Parse(pattern, syntax.Perl)  // ← syntax.Parse #2
    ...
}
```

**対処案:**  
内部ヘルパー `parseAndCheck(pattern) (*syntax.Regexp, error)` を導入して一度だけパースする。

---

## #6 — `Build(nil)` の動作が未テスト・未文書化 🟡 ✅ 対応済み

**ファイル:** `api.go`, `api_test.go`

**問題:**  
`Build(nil)` は有効なインデックス（番兵のみ）を返すが、これが意図的な仕様かどうか godoc に記載がない。  
また `Count`、`Search`、`SALen` などがどのような値を返すかを検証するテストが存在しない。

**対処案:**
1. `Build` の godoc に「nil または空のスライスを渡すと番兵のみのインデックスを返す」と追記する
2. `Build(nil)` と `Build([]byte{})` の動作を検証するテストを追加する

**対応内容:**  
`Build` の godoc に nil / 空スライスの動作（番兵のみのインデックス、`TextLen` = 0、`SALen` = 1）を明記し、
`TestBuildNilAndEmpty` で `Build(nil)` と `Build([]byte{})` の `TextLen`・`SALen`・`Count`・`Search` を検証済み。

---

## #7 — CI/CD ワークフローが存在しない 🔴 ✅ 対応済み

**問題:**  
`.github/workflows/` ディレクトリがなく、プッシュや PR ごとにテストが自動実行されない。  
外部ライブラリとして公開するにあたり、常に `go test -race ./...` が通ることを保証する仕組みが必要です。

**対処案:**  
`.github/workflows/ci.yml` を追加する：

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go vet ./...
      - run: go test -race ./...
```

**対応内容:**  
`.github/workflows/ci.yml` を追加。push / pull_request ごとに `go vet ./...` と
`go test -race ./...` を実行する（Go バージョンは `go.mod` から取得）。

---

## #8 — セマンティックバージョンタグがない 🔴

**問題:**  
`git tag` がなく、`go get github.com/bgnori/bwt-wheelerlang-study@latest` が機能しない。  
また `pkg.go.dev` へのインデックス登録も初回タグが必要です。

**対処案:**
```bash
git tag v0.1.0
git push origin v0.1.0
```

その後 `https://pkg.go.dev/github.com/bgnori/bwt-wheelerlang-study` を開くか、  
`GOPROXY=https://proxy.golang.org go get github.com/bgnori/bwt-wheelerlang-study@v0.1.0`  
を実行して Go プロキシにインデックス登録を促す。

---

## #9 — TUI 依存関係がライブラリ利用者の `go.mod` に現れる 🟢

**ファイル:** `go.mod`

**問題:**  
`go.mod` の direct 依存に `github.com/charmbracelet/bubbletea`、`github.com/charmbracelet/bubbles` が含まれる。  
これらは `cmd/textindex` のみが使用するが、ライブラリとして `go get` した利用者の `go mod tidy` 出力に余分な依存が現れる可能性があります。

**対処案（将来的な検討）:**  
`cmd/textindex` を独立したモジュール（`cmd/textindex/go.mod`）に切り出す、  
または `go.work` ワークスペースを使う構成に移行する。  
現状のメリット（単一リポジトリでの開発のしやすさ）と天秤にかけて判断すること。

---

## #10 — モジュールパスに "study" が含まれており外部ライブラリ名として不適切 🟢

**ファイル:** `go.mod`, `README.md`

**問題:**  
モジュールパス `github.com/bgnori/bwt-wheelerlang-study` には元々の「勉強用」という文脈の名称が含まれている。  
外部ライブラリとして広く公開する場合、利用者が長いパスを記述する必要があり、見た目が不自然です。

**候補:**

| 候補 | モジュールパス | 特徴 |
|------|----------------|------|
| `go-fmindex` | `github.com/bgnori/go-fmindex` | 汎用的・発見しやすい |
| `bwtsearch` | `github.com/bgnori/bwtsearch` | 既存のパッケージ名・バイナリ名と一致 |
| `go-starfree` | `github.com/bgnori/go-starfree` | 差別化特徴（星なし正規表現）を強調 |
| `fmsearch` | `github.com/bgnori/fmsearch` | 短く実用的 |
| `wheelerindex` | `github.com/bgnori/wheelerindex` | Wheeler グラフとの理論的つながりを強調 |

**注意:**  
モジュールパスの変更は後方互換性を破壊するため、`v1.0.0` タグを打つ前（現時点）に実施するのが最善です。

---

## #11 — `BuildStdlibFromFiles` のセパレータに `0x00` を渡してもエラーにならない 🔴 ✅ 対応済み

**ファイル:** `stdlib_index.go`

**問題:**  
`BuildFromFiles`・`BuildBiFromFiles` には 0x00 バイトの検証ループが追加されているが、
`BuildStdlibFromFiles` には同様のチェックが存在しない。

```go
// BuildStdlibFromFiles — セパレータに 0x00 を渡してもそのままテキストに連結される
BuildStdlibFromFiles(texts, []byte{0})  // ← panic しない / エラーにならない
```

StdlibIndex は Go 標準ライブラリの `suffixarray.Index` を使用するため FM-index の番兵問題は
直接は発生しないが、0x00 バイトが混入したテキストで構築されたインデックスは
検索結果が予測困難になり、ドキュメントとの一貫性が損なわれる。

**対処案:**  
`BuildFromFiles` と同様に 0x00 チェックを追加し、含まれる場合は panic する。

```go
for _, b := range separator {
    if b == 0x00 {
        panic("bwtsearch: separator must not contain 0x00")
    }
}
```

または、StdlibIndex はバイト 0x00 を特別扱いしない旨を godoc に明記する。

**対応内容:**  
`BuildStdlibFromFiles` に `BuildFromFiles` と同一の 0x00 検証ループを追加し、godoc に panic 条件を明記。
`TestBuildStdlibFromFilesPanicsOnNullSeparator` で検証済み。

---

## #12 — `BiIndex` / `StdlibIndex` のメソッドに nil レシーバーチェックがない 🔴 ✅ 対応済み

**ファイル:** `biindex.go`, `stdlib_index.go`

**問題:**  
`Index` の `Search`・`Append`・`WriteTo` には nil チェックが追加されているが（Issue #3 の部分対応）、
`BiIndex` と `StdlibIndex` の全メソッドには nil レシーバーチェックが存在しない。

```go
var idx *bwtsearch.BiIndex
idx.Count([]byte("abc"))         // → panic: nil pointer dereference
idx.WriteTo(os.Stdout)           // → panic: nil pointer dereference

var sidx *bwtsearch.StdlibIndex
sidx.Count([]byte("abc"))        // → panic: nil pointer dereference
```

Issue #3 と同根の問題だが、`BiIndex`・`StdlibIndex` は別ファイルで管理されているため
別 Issue として追跡する。

**対処案:**  
Issue #3 と同様に、以下のいずれかで統一する：
1. ゼロ値レシーバーに対してゼロ値を返す（`Count` → 0、`Locate` → nil など）
2. panic を "documented behavior" として godoc に明記する
3. `WriteTo` / `Save` には少なくともエラーを返す（破壊的変更なし）

**対応内容:**  
対処案 1 + 3 を採用。`BiIndex` の `TextLen`・`Count`・`Locate`・`ContextAround`・`FullInterval`・
`ExtendLeft`・`ExtendRight` と `StdlibIndex` の `TextLen`・`Count`・`Locate`・`ContextAround` は
nil レシーバーに対してゼロ値を返し、両者の `WriteTo`（および経由する `Save`）はエラーを返す。
`TestNilBiIndexZeroValues`・`TestNilStdlibIndexZeroValues` で検証済み。

---

## #13 — デシリアライズ時の長さフィールド検証が不十分 🔴 ⚠️ 部分対応

**ファイル:** `internal/fmindex/fmindex.go`, `biindex.go`, `stdlib_index.go`

**問題:**  
`ReadFrom`（`readCommonHeader`）、`ReadBiFrom`、`ReadStdlibFrom` は、ストリームから読み取った
長さフィールド（`n64`・`tlen`・`fwdLen`・`revLen` など）を検証せずに `make([]byte, n)` に渡している。

```go
// readCommonHeader — 検証なし
var tlen int64
binary.Read(br, binary.LittleEndian, &tlen)
idx.text = make([]byte, tlen)   // tlen < 0 → panic / 巨大値 → OOM
```

- 負の値が読み込まれた場合、`make` が `panic: makeslice: len out of range` で即座にクラッシュする
- 攻撃的に巨大な値（例: `1<<62`）を含む破損ファイルを `Load` すると、メモリを大量に確保しようとして OOM になり得る

信頼できないファイルを `Load` / `LoadBi` / `LoadStdlib` で読み込むユースケース（CLI でユーザー指定の
インデックスファイルを開くなど）では DoS ベクタになる。

**対処案:**  
各長さフィールド読み込み直後に検証を追加する：
1. 負の値なら即座にエラーを返す
2. 妥当な上限（残りストリームサイズが不明なため、たとえば `io.LimitReader` の併用や段階的読み込み）を検討する
3. 少なくとも `tlen < 0 || n64 < 0` チェックとエラー返却を全デシリアライザに追加する

**対応内容:**  
- `internal/fmindex` の `readCommonHeader`: `n64` が `1 <= n64 <= MaxInt/4`（SA バッファ `n*4` の
  オーバーフロー防止）であること、および `tlen == n64 - 1`（番兵の不変条件）であることを検証。
- `ReadBiFrom`: `fwdLen` / `revLen` が負の場合はエラーを返す。
- `ReadStdlibFrom`: `tlen` が負の場合はエラーを返す。

`TestReadFromRejectsCorruptLengths`・`TestReadBiFromRejectsNegativeLength`・
`TestReadStdlibFromRejectsNegativeLength` で検証済み。

**残課題:**
再レビュー（2026-07-31）では、負値と整数オーバーフローは拒否できる一方、入力ストリームの
残りサイズに対して長さが妥当かは検証されていないことを確認した。64 bit 環境の `MaxInt/4` は
実用上の上限として大きすぎ、`ReadBiFrom` と `ReadStdlibFrom` には正値の上限自体がない。
そのため、小さな破損ストリームに巨大な正の長さを記録すると、`io.ReadFull` より前の `make` で
過大なメモリ確保を試みる可能性が残る。読み込みバイト数の上限、既知サイズとの照合、または
段階的な制限付き読み込みが必要。

---

## #14 — `BiIndex.ExtendLeft` / `ExtendRight` が 1 ステップあたり最大 255 回の rank 呼び出しを行う 🟢

**ファイル:** `biindex.go`

**問題:**
`ExtendLeft` / `ExtendRight` は「c より小さい文字の出現数」を求めるために
`for b := 0; b < int(c); b++` で全文字を走査し、1 文字あたり 2 回の `OccCount` を呼んでいる。

```go
countLess := 0
for b := 0; b < int(c); b++ {
    countLess += f.OccCount(byte(b), bi.HiFwd) - f.OccCount(byte(b), bi.LoFwd)
}
```

1 回の拡張につき最大 510 回の rank クエリが発生するため、長いパターンの双方向検索や
seed-and-extend 系の近似検索では大きなオーバーヘッドになる。

**対処案（将来的な検討）:**  
- Wavelet Tree / Wavelet Matrix の `rangeCount`（区間内で c 未満の文字数を O(log σ) で数える操作）を
  内部 API として公開し、それを利用する
- 実際にインデックスに現れる文字のみ走査する（`AlphabetSize` ベースの文字リストを保持する）

---

## #15 — FM-index の入力本文に `0x00` が含まれても拒否されない 🔴

**ファイル:** `internal/fmindex/fmindex.go`, `api.go`, `biindex.go`

**問題:**
内部実装の godoc は `0x00` を一意な番兵として予約し、入力本文に含めてはならないとしているが、
`Build`・`BuildWithAlgorithm`・`BuildWithOptions`・`BuildBi*` は本文を検証していない。
`BuildFromFiles*` もセパレータだけを検証し、各本文内の `0x00` はそのまま連結する。
`Append` も追加文字列を検証しない。

本文に `0x00` があると番兵が一意でなくなり、suffix array 構築の前提が崩れて検索結果の正確性を
保証できない。CLI の `build` / `build-multi` からバイナリファイルを入力した場合にも発生し得る。

**対処案:**
- FM-index の全構築経路と `Append` で本文中の `0x00` を検出し、一貫した方法で拒否する
- 現行の構築 API は `error` を返さないため、当面は documented panic とし、将来の破壊的変更で
  エラーを返す API を検討する
- 単一入力、複数入力、双方向インデックス、Append の回帰テストを追加する

---

## #16 — 保存・復元後の `Append` で suffix-array アルゴリズムが保持されない ✅

**ファイル:** `internal/fmindex/fmindex.go`

**問題:**
`Append` は「構築時の suffix-array アルゴリズムを保持する」と説明されているが、永続化形式には
`algo` が保存されない。`readFromV1` と `readFromRebuildOcc` は読み込み時に常に
`AlgorithmDoubling` を設定するため、`AlgorithmSAIS` で構築したインデックスを保存・復元してから
`Append` すると、再構築処理が doubling に切り替わる。

検索結果の正確性には通常影響しないが、性能特性が API の説明と異なり、大規模データでは追加処理の
実行時間が大きく変わり得る。

**対処案:**
- 次の永続化フォーマットにアルゴリズム識別子を追加する
- 旧フォーマットは doubling として読み込む後方互換方針を明記する
- SA-IS で構築したインデックスの保存・復元・Append を検証するテストを追加する

**対応:** `FMIDX05`〜`FMIDX08` に構築アルゴリズムを保存し、読み込み後も `Append` が同じ
アルゴリズムを使用するようにした。旧 `FMIDX01`〜`FMIDX04` は doubling として読み込む。

---

## #17 — stdlib / BiIndex の検索結果が件数ちょうどでも truncated 扱いになる 🟡

**ファイル:** `cmd/bwtsearch/main.go`

**問題:**
`searchAny` は `StdlibIndex` と `BiIndex` について
`limit > 0 && len(pos) == limit` を truncated 判定に使う。この条件では、総件数が limit と
完全に一致して追加結果が存在しない場合にも `true` となり、CLI / TUI / Web UI に
「結果を切り詰めた」という誤った表示が出る。

FM-index の正規表現検索は `TotalCount > limit` で判定しており、バックエンド間で意味が一致しない。

**対処案:**
`Count(pattern) > limit` で判定するか、`limit+1` 件を取得して追加結果の有無を確認する。
総件数が `limit-1`、`limit`、`limit+1` のケースを各バックエンドでテストする。

---

## 対応状況まとめ

| # | タイトル | 優先度 | 状態 |
|---|----------|--------|------|
| 1 | 位置アンカーがサイレントに無視される | 🔴 High | ✅ 対応済み |
| 2 | `BuildFromFiles` のセパレータ検証なし | 🔴 High | ✅ 対応済み |
| 3 | `*Index` メソッドの nil チェック非一貫 | 🔴 High | ✅ 対応済み |
| 4 | Unicode 文字クラスがエラーなく 0 件になる | 🟡 Medium | ✅ 対応済み（ドキュメント明記） |
| 5 | `Search` が正規表現を 2 回パース | 🟡 Medium | ✅ 対応済み |
| 6 | `Build(nil)` が未テスト・未文書化 | 🟡 Medium | ✅ 対応済み |
| 7 | CI/CD ワークフローがない | 🔴 High | ✅ 対応済み |
| 8 | セマンティックバージョンタグがない | 🔴 High | 未対応 |
| 9 | TUI 依存が利用者の `go.mod` に現れる | 🟢 Low | 未対応 |
| 10 | モジュールパスに "study" が含まれる | 🟢 Low | 未対応 |
| 11 | `BuildStdlibFromFiles` の 0x00 セパレータ検証なし | 🔴 High | ✅ 対応済み |
| 12 | `BiIndex`/`StdlibIndex` のメソッドに nil レシーバーチェックなし | 🔴 High | ✅ 対応済み |
| 13 | デシリアライズ時の長さフィールド検証が不十分 | 🔴 High | ⚠️ 部分対応 |
| 14 | `BiIndex` 拡張が 1 ステップあたり O(σ) 回の rank 呼び出し | 🟢 Low | 未対応 |
| 15 | FM-index の入力本文に `0x00` が含まれても拒否されない | 🔴 High | 未対応 |
| 16 | 保存・復元後の `Append` で構築アルゴリズムが保持されない | 🟡 Medium | ✅ 対応済み |
| 17 | stdlib / BiIndex の検索結果が件数ちょうどでも truncated 扱いになる | 🟡 Medium | 未対応 |
