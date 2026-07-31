# shenwei356/bwt の近似検索（Hamming 距離 k-mismatch）実装解説

## 概要

[shenwei356/bwt](https://github.com/shenwei356/bwt) の `fmi` パッケージは、
完全一致だけでなく **Hamming 距離 k 以内の近似検索（k-mismatch search）** を
FM-index 上で直接実行できる。
API としては `fmi.Locate(query []byte, mismatches int)` と
`fmi.Match(query []byte, mismatches int)` の 2 関数が提供されている。

---

## 背景：FM-index の完全一致検索

まず完全一致検索の仕組みを確認する。
FM-index は BWT（Burrows-Wheeler Transform）から構築される索引で、
以下の 2 つのテーブルを中心に構成される。

| テーブル | 内容 |
|---------|------|
| `C[c]` | アルファベット `c` より辞書順で小さい文字が BWT のどの範囲に来るかを示すオフセット |
| `Occ[c][i]` | BWT の先頭 `i` 文字（0-based）中に `c` が何回出現するかの累積カウント |

完全一致検索では、クエリを末尾から 1 文字ずつ読み、

```
start = C[c] + Occ[c][start - 1]
end   = C[c] + Occ[c][end] - 1
```

という **後方検索（backward search）** を繰り返す。
`start > end` になった時点でその文字列はテキストに存在しない。

---

## k-mismatch 検索の基本戦略

完全一致の backward search を **部分的に緩める** ことで近似検索を実現する。
具体的には各ステップで「クエリの現在文字と同じ文字のみで backward search を進める」
代わりに、**アルファベット全体の文字で分岐を試みる**。

ミスマッチが発生した（クエリの文字と異なる文字で分岐した）場合は
残り許容ミスマッチ数を 1 減らし、探索を続ける。
許容ミスマッチ数が 0 になったら以降は完全一致のみで進める。

この方式は **Hamming 距離** 制約（挿入・削除なし、置換のみ）に対応した
近似 backward search であり、
文献では「k-mismatch backward search」と呼ばれる手法に相当する。

---

## 実装の詳細

### データ構造（`fmi.go`）

```go
type FMIndex struct {
    BWT         []byte    // Burrows-Wheeler 変換後の文字列
    SuffixArray []int     // テキストのサフィックス配列
    Alphabet    []byte    // テキストに出現するアルファベット
    CountOfLetters []int  // 各文字の出現頻度（長さ 128 のスライス）
    C   []int             // C テーブル（長さ 128 のスライス）
    Occ []*[]int32        // Occ テーブル（長さ 128 のスライス）
}
```

`C` と `Occ` は `map` ではなく固定長スライス（長さ 128 = ASCII 範囲）で
実装されており、文字をそのままインデックスとして使うことで
map のハッシュオーバーヘッドを排除している。

### 探索状態のスタック管理

```go
type sMatch struct {
    query      []byte  // 未処理のクエリ残り（末尾から読む）
    start, end int     // 現在の BWT 区間（0-based）
    mismatches int     // 残り許容ミスマッチ数
}
```

`Locate` 関数は `sMatch` のスタックを使って **深さ優先** で探索木を展開する。
初期状態は全 BWT 区間 `[0, n-1]` に対してクエリ全体をセットする。

### コアループの動作

```go
for !matches.Empty() {
    match = matches.Pop()
    query = match.query[0 : len(match.query)-1]  // 残り（末尾の 1 文字を除く）
    last  = match.query[len(match.query)-1]       // 現在処理する文字

    // 残りミスマッチが 0 なら完全一致のみ試みる
    if match.mismatches == 0 {
        letters = []byte{last}
    } else {
        letters = fmi.Alphabet  // 全アルファベットで分岐
    }

    for _, c = range letters {
        // backward search で区間を更新
        start = fmi.C[c] + Occ[c][match.start-1]
        end   = fmi.C[c] + Occ[c][match.end] - 1

        if start > end { continue }  // この分岐は不一致

        if len(query) == 0 {
            // クエリ末尾に到達 → SA からテキスト位置を取り出す
            for _, i := range fmi.SuffixArray[start : end+1] {
                locationsMap[i] = struct{}{}
            }
        } else {
            m = match.mismatches
            if c != last {          // ミスマッチが発生した
                m = match.mismatches - 1
            }
            matches.Put(sMatch{query: query, start: start, end: end, mismatches: m})
        }
    }
}
```

#### 重要な注意点

* `if match.mismatches > 1 { m = match.mismatches - 1 } else { m = 0 }`  
  というコードが実際には存在する（0 以下への負のアンダーフロー防止）。

* 同じ位置が複数の分岐から発見される可能性があるため、
  結果は `locationsMap`（`map[int]struct{}`）で重複を除去し、
  最後に `sort.Ints` でソートして返す。

* `Match` 関数は `Locate` と同じアルゴリズムだが、
  最初のヒットが見つかった時点で `true` を返して早期終了する
  軽量版として実装されている。

---

## アルゴリズムの計算量

| フェーズ | 計算量 |
|---------|--------|
| インデックス構築 | O(n log n)（Go 標準 `index/suffixarray` ＝ DC3/Skew 法） |
| 完全一致検索 | O(\|P\|)（backward search） |
| k-mismatch 検索 | O(\|Σ\|^k × \|P\|)（最悪ケース） |
| Occ ルックアップ | O(1)（完全プレフィックス和配列） |

ここで \|Σ\| はアルファベットサイズ、\|P\| はクエリ長、k はミスマッチ数。
バイオインフォマティクスの典型的なユースケース（DNA: \|Σ\|=4、タンパク質: \|Σ\|=20）
では k が小さいため実用的な速度で動作する。

---

## 動作例

`fmi_test.go` から抜粋したテストケースを用いて挙動を示す。

```
テキスト: "acctatac"
クエリ:   "tac"
k=0 → [5]         （"tac" が 1 箇所）
k=1 → [3, 5]      （"tat" と "tac" がそれぞれ 1 mismatch 以内）
```

```
テキスト: "acctatac"
クエリ:   "caa"
k=2 → [1, 2, 3, 4, 5]   （2 文字まで違っていてよいため多数ヒット）
k=3 → [0, 1, 2, 3, 4, 5] （3 文字まで許容するとほぼ全位置がヒット）
```

---

## 実装上の制限

| 制限 | 詳細 |
|------|------|
| **ASCII のみ** | `C` および `Occ` のスライス長が 128 に固定されている。0x80 以上のバイト（UTF-8 マルチバイト文字）はパニックを引き起こす |
| **Hamming 距離のみ** | 挿入・削除（編集距離 / Levenshtein）は非対応。置換のみ |
| **永続化なし** | インデックスをファイルに保存・読み込みする機能はない |
| **Go reflection 依存** | `bwt.SuffixArray` が Go 標準ライブラリの内部フィールドを reflection で取り出しており、Go バージョン変更で壊れる可能性がある |

---

## 外部情報源

### アルゴリズム（FM-index・近似検索）

| タイトル | 種別 | URL |
|---------|------|-----|
| Ferragina, P. & Manzini, G. (2000) "Opportunistic Data Structures with Applications" | 原論文（FM-index 提案） | <https://dl.acm.org/doi/10.1145/1073814.1073840> |
| Langmead, B. et al. (2009) "Ultrafast and memory-efficient alignment of short DNA sequences to the human genome" (Bowtie) | 応用論文（BWT による short-read 近似アライメント） | <https://genomebiology.biomedcentral.com/articles/10.1186/gb-2009-10-3-r25> |
| Li, H. & Durbin, R. (2009) "Fast and accurate short read alignment with Burrows-Wheeler Aligner" (BWA) | 応用論文（Hamming/Levenshtein 近似検索の実装比較） | <https://academic.oup.com/bioinformatics/article/25/14/1754/225615> |
| Navarro, G. & Mäkinen, V. (2007) "Compressed Full-Text Indexes" | サーベイ論文（FM-index の理論的背景） | <https://dl.acm.org/doi/10.1145/1233912.1233987> |
| Lam, T.W. et al. (2009) "Compressed Indexing and Local Alignment of DNA" (BWA 2-way BWT) | 論文（2-way backward search による近似検索） | <https://dl.acm.org/doi/10.1145/1542362.1542425> |

### 教育・解説リソース

| タイトル | 種別 | URL |
|---------|------|-----|
| Ben Langmead — "Lecture: BWT and FM-index" (Johns Hopkins) | 講義スライド PDF | <https://www.cs.jhu.edu/~langmea/resources/lecture_notes/10_bwt_and_fm_index.pdf> |
| Ben Langmead — "Lecture: FM-index Approximate Matching" (Johns Hopkins) | 講義スライド PDF | <https://www.cs.jhu.edu/~langmea/resources/lecture_notes/11_approximate_matching.pdf> |
| Coursera — "Algorithms for DNA Sequencing" (Ben Langmead) | 動画講義 | <https://www.coursera.org/learn/dna-sequencing> |
| Wikipedia — "FM-index" | 入門説明 | <https://en.wikipedia.org/wiki/FM-index> |
| Wikipedia — "Burrows–Wheeler transform" | BWT の基礎 | <https://en.wikipedia.org/wiki/Burrows%E2%80%93Wheeler_transform> |

### shenwei356/bwt ソースコード

| ファイル | 内容 |
|---------|------|
| [`fmi/fmi.go`](https://github.com/shenwei356/bwt/blob/master/fmi/fmi.go) | `Locate` / `Match` の本体実装、`computeC`、`computeOccurrence` |
| [`fmi/fmi_test.go`](https://github.com/shenwei356/bwt/blob/master/fmi/fmi_test.go) | k-mismatch を含むテストケース一覧 |
| [`bwt.go`](https://github.com/shenwei356/bwt/blob/master/bwt.go) | BWT 変換・サフィックス配列構築 |

---

## 関連ドキュメント（本リポジトリ内）

- [`docs/library_comparison.md`](library_comparison.md) — shenwei356/bwt を含む 3 ライブラリの機能・性能比較
- [`docs/performance_report.md`](performance_report.md) — ベンチマーク結果
