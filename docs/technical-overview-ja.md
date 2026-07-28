# BWT-WheelerLang-Study 実装解説

## はじめに

このリポジトリは「全文検索インデックス」の仕組みを Go で一から実装したものです。

`grep` などの素朴な全文検索はテキストを先頭から末尾まで走査するため、大きなテキストだと遅くなります。このリポジトリで実装している **FM-index** を使うと、テキストを事前に処理してインデックスを作ることで、**パターンの長さに比例した高速な検索**が可能になります。

大まかな構成は次の 3 層です：

```
starfree  (正規表現レイヤー)
   ↓
fmindex   (全文検索インデックス)
   ↓
bitvector (簡潔データ構造)
```

---

## 1. ビットベクトル（`internal/bitvector`）

### 役割

FM-index が「ある文字が BWT の先頭 i 文字に何回出てくるか」を高速に答えるための基礎データ構造です。

### 仕組み

ビットベクトルとは、各要素が 0 か 1 のみの配列です。`Rank1(i)` は「位置 0 から i-1 の範囲に 1 がいくつあるか」を返す操作で、これが O(1) で動くのがポイントです。

```
ビット列:  0 1 1 0 1 0 0 1 ...
Rank1(4) = 2  （先頭 4 ビット中、1 は 2 個）
```

64 ビット単語ごとに「先頭からの累積カウント」を保存しておき（`block` 配列）、クエリ時はブロック内の余りを CPU の **popcount（ビット数計算）** で補います。

```go
// ブロック境界まで: block[word]
// 残りのビット: popcount(data[word] & ((1<<bit)-1))
count := bv.block[word]
if bit > 0 {
    count += popcount(bv.data[word] & ((1 << bit) - 1))
}
```

popcount は Hamming Weight アルゴリズムで手動実装されています：

```go
func popcount(x uint64) int {
    x -= (x >> 1) & 0x5555555555555555
    x = (x & 0x3333333333333333) + ((x >> 2) & 0x3333333333333333)
    x = (x + (x >> 4)) & 0x0f0f0f0f0f0f0f0f
    return int((x * 0x0101010101010101) >> 56)
}
```

---

## 2. FM-index（`internal/fmindex`）

### FM-index とは

テキストから作成するインデックスで、以下を組み合わせた構造です：

| 要素 | 説明 |
|---|---|
| **BWT** (Burrows-Wheeler Transform) | テキストを変換して圧縮効率を上げたもの |
| **接尾辞配列 (SA)** | テキストの全接尾辞を辞書順に並べた配列 |
| **C 配列** | 各文字より辞書順で小さい文字が何個あるか |
| **Occ 配列** | BWT 中の各文字の出現頻度（ビットベクトルで実現） |

### BWT の仕組み

テキスト `T` にセンチネル文字（`\0`）を追加した後、**全接尾辞を辞書順に並べた行列の最終列**が BWT です。例えば `banana\0` に対して作ると次のようになります：

```
F 列（接尾辞の先頭）    L 列（= BWT）
\0banana            →   a
a\0banana           →   n
ana\0ban            →   n
anana\0b            →   a
banana\0            →   \0
na\0banan           →   a
nana\0ba            →   b
```

L 列（BWT）は `ann a\0ab` のように似た文字が近くにまとまる傾向があり、圧縮に適しています。

### 接尾辞配列の構築

実装では **Manber-Myers アルゴリズム（prefix-doubling）** を使っています。ランクを 1 文字 → 2 文字 → 4 文字 → … と 2 倍ずつ延ばしながらソートを繰り返し、O(n log² n) で構築します。

### 後方検索（Backward Search）

FM-index の本質です。パターン `P` を**右から左へ** 1 文字ずつ処理しながら、接尾辞配列上の「候補区間 [lo, hi)」を絞り込みます。

1 文字 `b` の処理式：

```
newLo = C[b] + Occ(b, lo)
newHi = C[b] + Occ(b, hi)
```

コードで確認：

```go
func (idx *Index) BackwardSearchStep(b byte, lo, hi int) (int, int) {
    newLo := idx.c[b] + idx.OccCount(b, lo)
    newHi := idx.c[b] + idx.OccCount(b, hi)
    return newLo, newHi
}
```

`lo >= hi` になったらパターンが存在しない、`hi - lo` がヒット件数です。

### Wheeler グラフとの対応

FM-index は「Wheeler グラフ」の圧縮表現でもあります：

| Wheeler グラフの概念 | FM-index での実現 |
|---|---|
| ノード | 接尾辞配列の各位置（F 列） |
| エッジのラベル | BWT の文字（L 列） |
| Wheeler 順序 | 接尾辞の辞書式順序 |
| ノードのラベル付け | C 配列 + Occ ビットベクトル |

後方検索は「Wheeler グラフ上の節点区間を 1 文字ずつ左に延長する操作」に対応します。

---

## 3. 星なし正規表現検索（`internal/starfree`）

### 星なし言語とは

通常の正規表現では `a*`（Kleene スター）や `a+`（1 回以上）が使えます。しかし FM-index の後方検索は「固定長のマッチを区間として表現する」ため、**長さが無制限になるパターンとの相性が悪い**のです。

そこで「星なし言語（star-free language）」を定義します。使えるのは：

| 演算 | 記号 | 説明 |
|---|---|---|
| リテラル | `hello` | 固定文字列 |
| 任意 1 文字 | `.` | 改行以外の 1 文字 |
| 文字クラス | `[abc]` | 指定文字のいずれか |
| 選択 | `a\|b` | どちらか |
| グループ | `(...)` | グループ化 |
| 省略可能 | `a?` | 0 or 1 回（= `a\|ε`） |
| 有界繰り返し | `a{2,5}` | n ～ m 回（有限） |

**使えないのは** `a*`、`a+`、`a{2,}` などの**無限繰り返し**です。

星なし言語は形式言語理論では**非周期的正規言語（aperiodic regular language）**と同値であることが知られており、カウンターフリー（counter-free）なオートマトンで受理される言語のクラスに対応します。

### Check（バリデーション）

`syntax.Parse` でパターンを AST（構文木）に変換し、禁止演算子ノードが含まれていないか再帰的に確認します：

```go
func checkNode(re *syntax.Regexp) error {
    switch re.Op {
    case syntax.OpStar:
        return &ViolationError{Op: "Kleene star (*)", SubExpr: re.String()}
    case syntax.OpPlus:
        return &ViolationError{Op: "one-or-more (+)", SubExpr: re.String()}
    case syntax.OpRepeat:
        if re.Max == -1 { // 無界
            return &ViolationError{...}
        }
    }
    for _, sub := range re.Sub {
        if err := checkNode(sub); err != nil {
            return err
        }
    }
    return nil
}
```

### Search（検索の実行）

正規表現の AST を再帰的にたどりながら、FM-index の区間 `[lo, hi)` を絞り込みます。

**連結（Concat）** は右から左へ処理（後方検索の方向に一致）：

```go
case syntax.OpConcat:
    intervals := []Interval{{lo, hi}}
    for i := len(re.Sub) - 1; i >= 0; i-- { // 右から左へ処理
        var next []Interval
        for _, iv := range intervals {
            next = append(next, evalRegex(idx, re.Sub[i], iv.Lo, iv.Hi)...)
        }
        intervals = mergeIntervals(next)
    }
```

**選択（Alternate）** は各枝の結果をまとめて区間マージ：

```go
case syntax.OpAlternate:
    var result []Interval
    for _, sub := range re.Sub {
        result = append(result, evalRegex(idx, sub, lo, hi)...)
    }
    return mergeIntervals(result)
```

**有界繰り返し（`{n,m}`）** は展開して処理します。まず `n` 回適用し、次に `n+1, n+2, …, m` 回の結果もすべて集めてマージします。

複数の区間が得られた場合は `mergeIntervals` で重複・隣接を統合します。

---

## 4. 永続化（シリアライズ）

FM-index のビルドには時間がかかるため、作ったインデックスをファイルに保存・読み込みできます。マジックバイト `"FMIDX01"` から始まるバイナリ形式で、テキスト・BWT・SA・C 配列・Occ ビットベクトルをすべてリトルエンディアンで書き出します。

---

## まとめ

| コンポーネント | 計算量 | 役割 |
|---|---|---|
| `bitvector.Rank1` | O(1) | Occ クエリの高速化 |
| `fmindex.Build` | O(n log² n) | インデックス構築 |
| `fmindex.BackwardSearch` | O(\|P\|) | パターン検索 |
| `starfree.Check` | O(\|AST\|) | 星なし検証 |
| `starfree.Search` | O(\|P\| × Σ) | 正規表現検索 |

FM-index は「テキストを圧縮しながら高速に検索できる」という優れた性質を持ちます。このリポジトリはそれを Wheeler グラフの理論的枠組みと対応付けつつ、ゼロから実装した学習用コードです。接尾辞配列・BWT・ランク演算・星なし言語という複数の概念が 1 本のコードの中で綺麗に組み合わさっているのが特徴です。

---

## 参考文献

### 書籍

1. **Dan Gusfield** (1997). *Algorithms on Strings, Trees, and Sequences: Computer Science and Computational Biology*. Cambridge University Press.  
   接尾辞配列・接尾辞木など文字列アルゴリズムの包括的な教科書。

2. **Paolo Ferragina, Raffaele Giancarlo** (eds.) (2009). *Combinatorial Pattern Matching*. Springer.  
   FM-index の原著論文を含む組み合わせパターンマッチングの講演録。

3. **Gonzalo Navarro** (2016). *Compact Data Structures: A Practical Approach*. Cambridge University Press.  
   簡潔データ構造（ビットベクトル・ウェーブレット木など）の体系的な教科書。FM-index の詳細な解説を含む。

### 論文

4. **Paolo Ferragina, Giovanni Manzini** (2000). "Opportunistic data structures with applications." *Proceedings of the 41st Annual Symposium on Foundations of Computer Science (FOCS)*, pp. 390–398.  
   FM-index の原著論文。BWT と接尾辞配列を組み合わせた圧縮インデックスを初めて提案。

5. **Udi Manber, Gene Myers** (1993). "Suffix arrays: A new method for on-line string searches." *SIAM Journal on Computing*, 22(5), pp. 935–948.  
   prefix-doubling による接尾辞配列構築アルゴリズムの原著論文。

6. **Michael Burrows, David J. Wheeler** (1994). *A Block-sorting Lossless Data Compression Algorithm*. Technical Report 124, Digital Equipment Corporation.  
   BWT（Burrows-Wheeler 変換）の原著技術報告書。

7. **Dominik Kempa, Nicola Prezza** (2019). "At the Roots of Dictionary Compression: String Attractors." *Proceedings of the 51st Annual ACM Symposium on Theory of Computing (STOC)*, pp. 827–840.

8. **Michael P. Schützenberger** (1965). "On finite monoids having only trivial subgroups." *Information and Control*, 8(2), pp. 190–194.  
   星なし言語と非周期的モノイドの同値性を示した定理の原著論文（Schützenberger の定理）。

9. **Robert McNaughton, Seymour Papert** (1971). *Counter-Free Automata*. MIT Press.  
   カウンターフリーオートマトンと星なし言語の同値性を証明した古典的著作。

10. **Travis Gagie, Giovanni Manzini, Jouni Sirén** (2020). "Wheeler Graphs: A Framework for BWT-Based Data Structures." *Theoretical Computer Science*, 820, pp. 1–20.  
    Wheeler グラフの理論的枠組みを定義し、FM-index との関係を体系化した論文。

### ウェブリソース

11. **Wikipedia — FM-index** (英語)  
    <https://en.wikipedia.org/wiki/FM-index>

12. **Wikipedia — Burrows-Wheeler transform** (英語)  
    <https://en.wikipedia.org/wiki/Burrows%E2%80%93Wheeler_transform>

13. **Wikipedia — Suffix array** (英語)  
    <https://en.wikipedia.org/wiki/Suffix_array>

14. **Wikipedia — Star-free language** (英語)  
    <https://en.wikipedia.org/wiki/Star-free_language>

15. **CP-Algorithms — Suffix Array** (英語)  
    <https://cp-algorithms.com/string/suffix-array.html>  
    prefix-doubling アルゴリズムの実装例と詳細解説。
