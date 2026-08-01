# FM-index ライブラリ比較

本ドキュメントでは、以下の3つのFM-index実装を比較する。

- **BWT-WheelerLang-Study**（本リポジトリ） — `github.com/bgnori/bwt-wheelerlang-study`
- **shenwei356/bwt** — `github.com/shenwei356/bwt`
- **rossmerr/fm-index** — `github.com/rossmerr/fm-index`

---

## 1. 基本情報

| 項目 | BWT-WheelerLang-Study | shenwei356/bwt | rossmerr/fm-index |
|---|---|---|---|
| 言語バージョン | Go 1.21 | Go 1.17 | Go 1.19 |
| ライセンス | MIT | MIT | 未確認 |
| 外部依存 | charmbracelet（TUIのみ） | **ゼロ**（stdlib のみ） | rossmerr/{bwt,wavelettree,bitvector} |
| スコープ | FM-index + Wheeler グラフ研究用ライブラリ + CLI | BWT変換 + FM-index ライブラリ | FM-index ライブラリ |

---

## 2. 内部データ構造

| 構造 | BWT-WheelerLang-Study | shenwei356/bwt/fmi | rossmerr/fm-index |
|---|---|---|---|
| **BWT格納** | `[]byte`（直接格納） | `[]byte`（直接格納） | Wavelet Tree（F列・L列の両方） |
| **Suffix Array** | `[]int32`（フル保存、int32節約） | `[]int`（フル保存） | Sampled SA（sampleRate毎に1つ） |
| **C配列** | `[256]int`（固定長256） | `[]int`（長さ128、ASCII限定） | F列のWavelet Tree Select演算で代用 |
| **Occ配列** | `[256]*bitvector.BitVector`（ビットベクトルランク） | `[]*[]int32`（長さ128、完全配列） | L列のWavelet Tree Rank演算で代用 |
| **アルファベット** | 8bit (0x00〜0xFF) 完全対応 | **7bit ASCII のみ**（配列サイズ128） | rune（UTF-16）ベース |
| **Rank演算** | **O(1)**（ビットベクトル + 事前構築ランク表） | O(1)（完全なプレフィックス和配列） | **O(n)**（線形スキャン、ランク表なし） |

---

## 3. SA（接尾辞配列）構築アルゴリズム

| | BWT-WheelerLang-Study | shenwei356/bwt | rossmerr/fm-index |
|---|---|---|---|
| **デフォルト** | Prefix-doubling法（Manber-Myers） O(n log² n) | Go標準 `index/suffixarray`（DC3/Skew） O(n log n) — reflectionで内部アクセス | 全サフィックス文字列を明示構築 + `sort.Strings` **O(n² log n)** |
| **代替** | **SA-IS法 O(n)**（`AlgorithmSAIS`オプション） | なし | なし |
| **備考** | ゼロコピー・番兵付き実装 | 標準ライブラリ活用だがreflection依存（将来のGoバージョンで壊れる可能性） | 実用上100KB超のテキストには不適 |

---

## 4. 公開API比較

| 機能 | BWT-WheelerLang-Study | shenwei356/bwt/fmi | rossmerr/fm-index |
|---|---|---|---|
| **インデックス構築** | `Build(text []byte)` / `BuildWithAlgorithm(...)` | `fmi.Transform(s []byte)` | `NewFMIndex(text string, opts...)` |
| **Count（件数カウント）** | `idx.Count(pattern []byte) int` | `len(fmi.Locate(query, 0))`（Count専用なし） | `idx.Count(pattern string) int` |
| **Locate（位置検索）** | `idx.Locate(pattern []byte, limit int) []int` | `fmi.Locate(query []byte, mismatches int) []int` | `idx.Locate(pattern string) []int` |
| **Extract（文字列復元）** | `idx.ContextAround(pos, patLen, ctxSize int) string` | なし | `idx.Extract(offset, length int) string` |
| **近似検索（k-mismatch）** | なし | **`fmi.Locate(query, k)`** — Hamming距離k以内 | なし |
| **正規表現検索** | **`Search(idx, pattern, limit)` — 星なし正規表現** | なし | なし |
| **BWT取得** | `idx.BWT() []byte` | `fmi.BWT` フィールド直接アクセス | Wavelet Tree経由（直接アクセス不可） |
| **永続化（保存/読み込み）** | **`idx.Save(path)` / `Load(path)`** バイナリ形式 | なし | なし |
| **Wheeler グラフ視覚化** | **`WheelerGraphMermaid(maxNodes int)` + CLIコマンド** | なし | なし |
| **インタラクティブUI** | **`textindex browse`** — TUI | なし | なし |

---

## 5. メモリ効率

| | BWT-WheelerLang-Study | shenwei356/bwt/fmi | rossmerr/fm-index |
|---|---|---|---|
| **SA** | n×4 bytes（int32） | n×8 bytes（int） | n/sampleRate×8 bytes（サンプリング） |
| **Occ** | ビットベクトル（\|Σ\|×n/8 bytes + ランク表） | \|Σ\|×n×4 bytes（完全prefixsum、int32） | Wavelet Tree（O(n log σ) bits） |
| **BWT** | n bytes | n bytes | Wavelet Tree内（圧縮） |
| **総評** | 中程度（ビットベクトルで効率化） | **最大**（完全Occ配列は大きいが高速アクセス） | サンプリングSA+Wavelet Treeで最も省メモリ志向だが、Rank O(n)が実用的には遅い |

---

## 6. 制限事項まとめ

| 制限 | BWT-WheelerLang-Study | shenwei356/bwt/fmi | rossmerr/fm-index |
|---|---|---|---|
| 文字セット | **8bit完全対応**（0x01〜0xFF）、0x00は番兵 | **7bit ASCII のみ**（≥128でpanicの危険） | rune（Unicode）対応、ETX(0x03)は番兵 |
| 日本語・マルチバイト | **UTF-8バイト列として対応** | **不可**（128以上のバイトでpanicの危険） | rune単位だが非ASCII rune追加でpanic危険 |
| 永続化 | **あり**（バイナリ形式、ファイル保存/読込） | **なし** | **なし** |
| 正規表現 | **星なし正規表現をサポート** | **なし**（リテラルと近似のみ） | **なし** |
| 近似検索 | **なし** | **あり**（Hamming距離 k-mismatch） | **なし** |
| 番兵文字 | 0x00を番兵として予約 | 0x00を番兵として予約 | ETX(0x03)を番兵として予約 |

---

## 7. 性能特性サマリ

| | BWT-WheelerLang-Study | shenwei356/bwt/fmi | rossmerr/fm-index |
|---|---|---|---|
| **構築速度** | O(n log² n)〜O(n)（アルゴリズム選択可） | O(n log n)（Go stdlib流用） | **O(n² log n)（実用上最も遅い）** |
| **Count/Locate速度** | O(\|P\|)、Rank O(1) | O(\|P\|)、Rank O(1)（完全配列） | O(\|P\|×n)（Rank O(n)のため） |
| **Locate近似** | N/A | O(\|Σ\|^k × \|P\|) | N/A |
| **メモリ（構築後）** | 中 | 大（完全Occ） | 小〜中（サンプリングSA）、ただしWTオーバーヘッド |

---

## 8. 総合評価

| 観点 | 推奨ライブラリ | 理由 |
|---|---|---|
| **日本語・マルチバイト文字の検索** | BWT-WheelerLang-Study | 8bitフル対応、UTF-8を正式サポート |
| **近似検索（タイポ・変異許容）** | shenwei356/bwt/fmi | k-mismatch（Hamming距離）をサポートする唯一の実装 |
| **インデックス保存・配布** | BWT-WheelerLang-Study | 唯一の永続化機能 |
| **正規表現検索** | BWT-WheelerLang-Study | 星なし正規表現（?\|\|{n,m}）をサポート |
| **大規模テキストの省メモリ** | rossmerr/fm-index（概念上） | サンプリングSA+Wavelet Treeだが、Rank O(n)で実用速度に課題 |
| **ゼロ依存・シンプル導入** | shenwei356/bwt | 外部依存なし、`go get`一発 |
| **学習・Wheeler グラフ理解** | BWT-WheelerLang-Study | Mermaid出力・TUIブラウザ・Wheeler グラフとの対応が明示的 |

---

## 9. 設計思想の違い

- **BWT-WheelerLang-Study**：Wheeler グラフとしてのFM-indexを教育・研究目的で実装。8bitフル対応・永続化・星なし正規表現・2種のSAアルゴリズムを備えた最も機能豊富な実装。
- **shenwei356/bwt**：実用的なバイオインフォマティクス向け（DNA/タンパク質配列の近似検索）を想定した設計。GoのsysCallをreflectionで活用する実用主義的実装。ASCII限定だが最も「動く」実装。
- **rossmerr/fm-index**：Wavelet Tree・Huffman符号・サンプリングSAといった理論的に洗練された構造を採用しているが、BitVectorのRankがO(n)のため、理論と実装速度が乖離している研究志向の実装。
